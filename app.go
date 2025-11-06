package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

type FileFilter struct {
	Pattern string `json:"pattern"`
	Name    string `json:"name"`
}

func (a *App) SelectFile(filters []FileFilter) (string, error) {
	filterOptions := make([]wailsruntime.FileFilter, len(filters))
	for i, filter := range filters {
		filterOptions[i] = wailsruntime.FileFilter{
			Pattern:     filter.Pattern,
			DisplayName: filter.Name,
		}
	}

	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:   "Select File",
		Filters: filterOptions,
	})
}

func (a *App) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

type RequestData struct {
	Method  string            `json:"method"`
	Url     string            `json:"url"`
	Data    string            `json:"data"`
	Cookie  string            `json:"cookie"`
	Headers map[string]string `json:"headers"`
}

type ResponseData struct {
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Data    []byte              `json:"data"`
}

func (a *App) Request(payload RequestData) (*ResponseData, error) {
	client := &http.Client{}
	req, err := http.NewRequest(payload.Method, payload.Url, strings.NewReader(payload.Data))
	if err != nil {
		return &ResponseData{Status: 0, Headers: nil, Data: []byte(`{"error": "` + err.Error() + `"}`)}, nil
	}

	if payload.Cookie != "" {
		req.Header.Add("Cookie", payload.Cookie)
	}

	for key, value := range payload.Headers {
		req.Header.Add(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return &ResponseData{Status: 0, Headers: nil, Data: []byte(`{"error": "` + err.Error() + `"}`)}, nil
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ResponseData{Status: 0, Headers: nil, Data: []byte(`{"error": "` + err.Error() + `"}`)}, nil
	}

	headers := make(map[string][]string)
	maps.Copy(headers, resp.Header)

	return &ResponseData{
		Status:  resp.StatusCode,
		Headers: headers,
		Data:    data,
	}, nil
}

func (a *App) Paint(payload RequestData) (*ResponseData, error) {
	// 1) Giải mã JSON thành [][]int với định dạng [[x,y,r,g,b,a], ...]
	var rawData [][]int
	if err := json.Unmarshal([]byte(payload.Data), &rawData); err != nil {
		return &ResponseData{Status: 0, Headers: nil, Data: []byte(`{"error": "` + err.Error() + `"}`)}, nil
	}

	// 2) Pack thành buffer nhị phân: mỗi điểm 8 byte (x:2, y:2, r:1, g:1, b:1, a:1)
	const bytesPerPoint = 8
	packed := make([]byte, len(rawData)*bytesPerPoint)
	offset := 0

	for i := 0; i < len(rawData); i++ {
		point := rawData[i]
		// Kỳ vọng point có đúng 6 phần tử: [x,y,r,g,b,a]
		if len(point) != 6 {
			return &ResponseData{Status: 0, Headers: nil, Data: []byte(`{"error": "invalid point length"}`)}, nil
		}

		x := uint16(point[0])
		y := uint16(point[1])
		r := uint8(point[2])
		g := uint8(point[3])
		b := uint8(point[4])
		alpha := uint8(point[5])

		// Big Endian cho x,y
		binary.BigEndian.PutUint16(packed[offset+0:], x)
		binary.BigEndian.PutUint16(packed[offset+2:], y)

		// r,g,b,a mỗi cái 1 byte
		packed[offset+4] = byte(r)
		packed[offset+5] = byte(g)
		packed[offset+6] = byte(b)
		packed[offset+7] = byte(alpha)

		offset += bytesPerPoint
	}

	// 3) Gửi buffer
	client := &http.Client{}
	req, err := http.NewRequest(payload.Method, payload.Url, bytes.NewReader(packed))
	if err != nil {
		return &ResponseData{Status: 0, Headers: nil, Data: []byte(`{"error": "` + err.Error() + `"}`)}, nil
	}

	if payload.Cookie != "" {
		req.Header.Add("Cookie", payload.Cookie)
	}

	for key, value := range payload.Headers {
		req.Header.Add(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return &ResponseData{Status: 0, Headers: nil, Data: []byte(`{"error": "` + err.Error() + `"}`)}, nil
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ResponseData{Status: 0, Headers: nil, Data: []byte(`{"error": "` + err.Error() + `"}`)}, nil
	}

	headers := make(map[string][]string)
	maps.Copy(headers, resp.Header)

	return &ResponseData{
		Status:  resp.StatusCode,
		Headers: headers,
		Data:    data,
	}, nil
}

func (a *App) WriteSettings(settings string) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	exeDir := filepath.Dir(exePath)
	path := filepath.Join(exeDir, "settings.json")
	return os.WriteFile(path, []byte(settings), 0644)
}

func (a *App) ReadSettings() (*string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, err
	}

	exeDir := filepath.Dir(exePath)
	path := filepath.Join(exeDir, "settings.json")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	strData := string(data)
	return &strData, nil
}

func (a *App) OpenURL(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
