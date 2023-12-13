package models

type FileContent struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type BodyPistonRequest struct {
	Language           string        `json:"language"`
	Version            string        `json:"version"`
	Files              []FileContent `json:"files"`
	Stdin              string        `json:"stdin"`
	CompileTimeout     int           `json:"compile_timeout"`
	RunTimeout         int           `json:"run_timeout"`
	CompileMemoryLimit int           `json:"compile_memory_limit"`
	RunMemoryLimit     int           `json:"run_memory_limit"`
	Args               []string      `json:"args"`
}

type RunPistonResponse struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Code   int32  `json:"code"`
	Output string `json:"output"`
}

type PistonResponse struct {
	Run      RunPistonResponse `json:"run"`
	Language string            `json:"language"`
	Version  string            `json:"version"`
}
