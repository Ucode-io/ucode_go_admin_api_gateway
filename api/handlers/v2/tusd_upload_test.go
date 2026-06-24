package v2

import "testing"

func TestMovieContentType(t *testing.T) {
	tests := map[string]struct {
		fileType string
		want     string
	}{
		"missing metadata":   {fileType: "", want: "video/mp4"},
		"invalid metadata":   {fileType: "not-a-mime-type", want: "video/mp4"},
		"non-video metadata": {fileType: "application/pdf", want: "video/mp4"},
		"mp4 metadata":       {fileType: "video/mp4", want: "video/mp4"},
		"webm metadata":      {fileType: "video/webm", want: "video/webm"},
		"metadata params":    {fileType: "video/mp4; codecs=avc1.4D401F", want: "video/mp4"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := movieContentType(test.fileType); got != test.want {
				t.Fatalf("movieContentType(%q) = %q, want %q", test.fileType, got, test.want)
			}
		})
	}
}
