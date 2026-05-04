package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
)

const (
	unsplashBase    = "https://api.unsplash.com"
	unsplashTimeout = 10 * time.Second
	imgCacheMax     = 200
)

// UnsplashPhoto is one result from the Unsplash search API.
type UnsplashPhoto struct {
	ID           string
	URLHero      string // 1600×900
	URLCard      string // 800×600
	URLThumb     string // 400×300
	Photographer string
	PhotoPage    string
}

// UnsplashResult holds the pre-fetched image pool for a project.
type UnsplashResult struct {
	Photos []UnsplashPhoto
}

var (
	imgCacheMu  sync.Mutex
	imgCacheMap = make(map[string]*UnsplashResult, imgCacheMax)
)

func imgCacheKey(keywords []string) string {
	sorted := make([]string, len(keywords))
	copy(sorted, keywords)
	sort.Strings(sorted)
	return strings.Join(sorted, "|")
}

// ExtractImageKeywords derives up to 3 search terms from an ArchitectPlan.
// Strategy: project name significant words first, then unique table labels.
func ExtractImageKeywords(plan *models.ArchitectPlan) []string {
	skipWord := map[string]bool{
		"the": true, "for": true, "and": true, "with": true, "app": true,
		"panel": true, "system": true, "admin": true, "platform": true,
		"pro": true, "plus": true, "your": true, "our": true,
	}
	skipTable := map[string]bool{
		"users": true, "user": true, "settings": true, "setting": true,
		"logs": true, "log": true, "roles": true, "role": true,
	}

	seen := map[string]bool{}
	var result []string

	add := func(w string) {
		w = strings.ToLower(strings.Trim(w, ".,;:!?\"'-_()"))
		if len(w) > 3 && !skipWord[w] && !seen[w] {
			seen[w] = true
			result = append(result, w)
		}
	}

	for _, w := range strings.Fields(plan.ProjectName) {
		add(w)
	}
	for _, t := range plan.Tables {
		for _, part := range strings.Fields(t.Label) {
			if !skipTable[strings.ToLower(strings.TrimSpace(t.Label))] {
				add(part)
			}
		}
	}

	if len(result) > 3 {
		result = result[:3]
	}
	return result
}

// unsplashRaw is the minimal Unsplash /search/photos response shape.
type unsplashRaw struct {
	ID   string `json:"id"`
	URLs struct {
		Raw string `json:"raw"`
	} `json:"urls"`
	User  struct{ Name string `json:"name"` }  `json:"user"`
	Links struct{ HTML string `json:"html"` } `json:"links"`
}

// imgixURL appends Imgix sizing params to an Unsplash raw URL.
// The raw URL may already contain query params (ixid, ixlib), so we join with &.
func imgixURL(raw string, w, h int) string {
	sep := "&"
	if !strings.Contains(raw, "?") {
		sep = "?"
	}
	return fmt.Sprintf("%s%sw=%d&h=%d&fit=crop&auto=format&q=80", raw, sep, w, h)
}

// searchUnsplash calls GET /search/photos and returns up to count photos.
func searchUnsplash(ctx context.Context, accessKey, query string, count int) ([]UnsplashPhoto, error) {
	if count > 30 {
		count = 30
	}
	params := url.Values{
		"query":          {query},
		"per_page":       {fmt.Sprintf("%d", count)},
		"orientation":    {"landscape"},
		"content_filter": {"high"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		unsplashBase+"/search/photos?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Client-ID "+accessKey)

	client := &http.Client{Timeout: unsplashTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("unsplash %d: %s", resp.StatusCode, string(b))
	}

	var data struct {
		Results []unsplashRaw `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	out := make([]UnsplashPhoto, 0, len(data.Results))
	for _, p := range data.Results {
		out = append(out, UnsplashPhoto{
			ID:           p.ID,
			URLHero:      imgixURL(p.URLs.Raw, 1600, 900),
			URLCard:      imgixURL(p.URLs.Raw, 800, 600),
			URLThumb:     imgixURL(p.URLs.Raw, 400, 300),
			Photographer: p.User.Name,
			PhotoPage:    p.Links.HTML,
		})
	}
	log.Printf("[unsplash] search %q → %d photos", query, len(out))
	return out, nil
}

// fetchForPlan fetches 8 photos (2 hero + 6 card) for the given keywords.
// Falls back to first keyword alone if the primary joined query returns fewer than needed.
func fetchForPlan(ctx context.Context, accessKey string, keywords []string) (*UnsplashResult, error) {
	key := imgCacheKey(keywords)

	imgCacheMu.Lock()
	if cached, ok := imgCacheMap[key]; ok {
		imgCacheMu.Unlock()
		return cached, nil
	}
	imgCacheMu.Unlock()

	const needed = 8
	query := strings.Join(keywords, " ")

	photos, err := searchUnsplash(ctx, accessKey, query, needed)
	if err != nil {
		return nil, err
	}
	// Fallback if primary query was too specific and returned few results
	if len(photos) < needed && len(keywords) > 1 {
		extra, extraErr := searchUnsplash(ctx, accessKey, keywords[0], needed-len(photos))
		if extraErr == nil {
			photos = append(photos, extra...)
		}
	}

	result := &UnsplashResult{Photos: photos}

	imgCacheMu.Lock()
	if len(imgCacheMap) >= imgCacheMax {
		imgCacheMap = make(map[string]*UnsplashResult, imgCacheMax) // simple eviction
	}
	imgCacheMap[key] = result
	imgCacheMu.Unlock()

	return result, nil
}

// FetchImagePoolBlock fetches contextual Unsplash images for the plan and formats them
// as a structured prompt block that Claude can consume directly.
//
// Returns "" (empty string) if:
//   - accessKey is missing (UNSPLASH_ACCESS_KEY not set)
//   - the API call fails for any reason
//
// In both cases generation continues unaffected — Claude falls back to VERIFIED PHOTO LIBRARY.
func FetchImagePoolBlock(ctx context.Context, accessKey string, plan *models.ArchitectPlan) string {
	if accessKey == "" {
		return ""
	}
	keywords := ExtractImageKeywords(plan)
	if len(keywords) == 0 {
		return ""
	}

	result, err := fetchForPlan(ctx, accessKey, keywords)
	if err != nil {
		log.Printf("[unsplash] non-fatal fetch error (falling back to library): %v", err)
		return ""
	}
	if len(result.Photos) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("════════════════════════════════════════\n")
	sb.WriteString("IMAGE POOL — USE THESE EXACT URLs\n")
	sb.WriteString("════════════════════════════════════════\n")
	fmt.Fprintf(&sb, "Contextual Unsplash images pre-fetched for \"%s\".\n", plan.ProjectName)
	sb.WriteString("Use these INSTEAD OF the VERIFIED PHOTO LIBRARY below.\n")
	sb.WriteString("NEVER invent photo IDs. NEVER use placeholder.com / picsum.photos.\n\n")

	sb.WriteString("HERO (1600×900) — hero sections, full-bleed banners, page backgrounds:\n")
	for i, p := range result.Photos {
		if i >= 2 {
			break
		}
		fmt.Fprintf(&sb, "  • %s\n", p.URLHero)
		if p.Photographer != "" {
			fmt.Fprintf(&sb, "    by %s\n", p.Photographer)
		}
	}

	sb.WriteString("\nCARD (800×600) — feature cards, product/service sections, team photos:\n")
	for i, p := range result.Photos {
		if i < 2 || i >= 8 {
			continue
		}
		fmt.Fprintf(&sb, "  • %s\n", p.URLCard)
	}

	sb.WriteString("\nTHUMB (400×300) — table row images, list thumbnails, small cards:\n")
	for i, p := range result.Photos {
		if i >= 4 {
			break
		}
		fmt.Fprintf(&sb, "  • %s\n", p.URLThumb)
	}

	sb.WriteString("\nIf you need more images than the pool provides: reuse pool URLs with different size params.\n")
	sb.WriteString("Always set descriptive alt text on every <img>.\n")
	sb.WriteString("════════════════════════════════════════\n")

	return sb.String()
}
