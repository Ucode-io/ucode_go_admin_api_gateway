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

var unsplashClient = &http.Client{Timeout: unsplashTimeout}

type unsplashPhoto struct {
	URLHero      string // 1600×900
	URLCard      string // 800×600
	URLThumb     string // 400×300
	Photographer string
}

var (
	imgCacheMu  sync.Mutex
	imgCacheMap = make(map[string][]unsplashPhoto, imgCacheMax)
)

func imgCacheKey(keywords []string) string {
	sorted := make([]string, len(keywords))
	copy(sorted, keywords)
	sort.Strings(sorted)
	return strings.Join(sorted, "|")
}

// imgixURL appends Imgix sizing params to an Unsplash raw URL.
func imgixURL(raw string, w, h int) string {
	sep := "&"
	if !strings.Contains(raw, "?") {
		sep = "?"
	}
	return fmt.Sprintf("%s%sw=%d&h=%d&fit=crop&auto=format&q=80", raw, sep, w, h)
}

// extractKeywords returns the image_keywords set by the Architect (max 4).
func extractKeywords(plan *models.ArchitectPlan) []string {
	kw := plan.ImageKeywords
	if len(kw) > 4 {
		kw = kw[:4]
	}
	return kw
}

// searchUnsplash calls GET /search/photos and returns up to count photos.
func searchUnsplash(ctx context.Context, accessKey, query string, count int) ([]unsplashPhoto, error) {
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

	resp, err := unsplashClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("unsplash %d: %s", resp.StatusCode, string(b))
	}

	var data struct {
		Results []struct {
			URLs  struct{ Raw string `json:"raw"` }  `json:"urls"`
			User  struct{ Name string `json:"name"` } `json:"user"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	out := make([]unsplashPhoto, 0, len(data.Results))
	for _, p := range data.Results {
		out = append(out, unsplashPhoto{
			URLHero:      imgixURL(p.URLs.Raw, 1600, 900),
			URLCard:      imgixURL(p.URLs.Raw, 800, 600),
			URLThumb:     imgixURL(p.URLs.Raw, 400, 300),
			Photographer: p.User.Name,
		})
	}
	log.Printf("[unsplash] search %q → %d photos", query, len(out))
	return out, nil
}

// fetchPhotos fetches up to 12 photos for the keywords with a single-keyword fallback.
func fetchPhotos(ctx context.Context, accessKey string, keywords []string) ([]unsplashPhoto, error) {
	key := imgCacheKey(keywords)

	imgCacheMu.Lock()
	if cached, ok := imgCacheMap[key]; ok {
		imgCacheMu.Unlock()
		return cached, nil
	}
	imgCacheMu.Unlock()

	const needed = 12
	photos, err := searchUnsplash(ctx, accessKey, strings.Join(keywords, " "), needed)
	if err != nil {
		return nil, err
	}
	// If the combined query was too specific, top up with the first keyword alone
	if len(photos) < needed && len(keywords) > 1 {
		extra, extraErr := searchUnsplash(ctx, accessKey, keywords[0], needed-len(photos))
		if extraErr == nil {
			photos = append(photos, extra...)
		}
	}

	imgCacheMu.Lock()
	if len(imgCacheMap) >= imgCacheMax {
		imgCacheMap = make(map[string][]unsplashPhoto, imgCacheMax)
	}
	imgCacheMap[key] = photos
	imgCacheMu.Unlock()

	return photos, nil
}

// ImagePoolResult is returned by FetchImagePool.
type ImagePoolResult struct {
	Block    string   // formatted prompt block to append to apiConfig; empty on failure
	Keywords []string // search terms that were used
	Count    int      // number of photos fetched
	Err      error    // non-nil when the API call failed; generation continues either way
}

// FetchImagePool fetches contextual Unsplash images for the plan and formats them
// as a prompt block for Claude. Always safe to call — Err is informational only.
func FetchImagePool(ctx context.Context, accessKey string, plan *models.ArchitectPlan) ImagePoolResult {
	keywords := extractKeywords(plan)
	log.Printf("[unsplash] project=%q keywords=%v", plan.ProjectName, keywords)

	if len(keywords) == 0 {
		return ImagePoolResult{Err: fmt.Errorf("no keywords extracted from plan")}
	}

	photos, err := fetchPhotos(ctx, accessKey, keywords)
	if err != nil {
		log.Printf("[unsplash] API error: %v", err)
		return ImagePoolResult{Keywords: keywords, Err: err}
	}
	if len(photos) == 0 {
		return ImagePoolResult{Keywords: keywords, Err: fmt.Errorf("0 results for %v", keywords)}
	}

	log.Printf("[unsplash] ✅ %d photos fetched", len(photos))

	var sb strings.Builder
	fmt.Fprintf(&sb, "════════════════════════════════════════\n")
	fmt.Fprintf(&sb, "IMAGE POOL — USE THESE EXACT URLs\n")
	fmt.Fprintf(&sb, "════════════════════════════════════════\n")
	fmt.Fprintf(&sb, "Pre-fetched for \"%s\" · query: %s\n", plan.ProjectName, strings.Join(keywords, " "))
	fmt.Fprintf(&sb, "NEVER invent photo IDs. NEVER use placeholder.com / picsum.photos.\n\n")

	// Photos 0–2: hero
	fmt.Fprintf(&sb, "HERO (1600×900) — hero sections, full-bleed banners:\n")
	for i, p := range photos {
		if i >= 3 {
			break
		}
		fmt.Fprintf(&sb, "  • %s", p.URLHero)
		if p.Photographer != "" {
			fmt.Fprintf(&sb, "  [%s]", p.Photographer)
		}
		fmt.Fprintf(&sb, "\n")
	}

	// Photos 3–11: cards
	fmt.Fprintf(&sb, "\nCARD (800×600) — feature cards, section images:\n")
	for i, p := range photos {
		if i < 3 {
			continue
		}
		fmt.Fprintf(&sb, "  • %s\n", p.URLCard)
	}

	// Photos 0–5: thumbs (different size, same contextual photos)
	fmt.Fprintf(&sb, "\nTHUMB (400×300) — table rows, small cards, list images:\n")
	for i, p := range photos {
		if i >= 6 {
			break
		}
		fmt.Fprintf(&sb, "  • %s\n", p.URLThumb)
	}

	fmt.Fprintf(&sb, "════════════════════════════════════════\n")

	return ImagePoolResult{
		Block:    sb.String(),
		Keywords: keywords,
		Count:    len(photos),
	}
}
