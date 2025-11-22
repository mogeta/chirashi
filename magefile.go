//go:build mage

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	ProjectName = "chirashi"
	BuildDir    = "build"
	WebBuildDir = "build/web"
	PublicDir   = "public"
)

// Build builds the Ebiten game for the current platform
func Build() error {
	fmt.Println("Building Ebiten game...")
	if err := os.MkdirAll(BuildDir, 0755); err != nil {
		return err
	}
	return sh.Run("go", "build", "-o", filepath.Join(BuildDir, getBinaryName()), ".")
}

// BuildRelease builds the Ebiten game with optimizations for release
func BuildRelease() error {
	fmt.Println("Building Ebiten game for release...")
	if err := os.MkdirAll(BuildDir, 0755); err != nil {
		return err
	}
	return sh.Run("go", "build", "-ldflags", "-s -w", "-o", filepath.Join(BuildDir, getBinaryName()), ".")
}

// BuildWindows builds the Ebiten game for Windows
func BuildWindows() error {
	fmt.Println("Building Ebiten game for Windows...")
	if err := os.MkdirAll(BuildDir, 0755); err != nil {
		return err
	}
	env := map[string]string{
		"GOOS":   "windows",
		"GOARCH": "amd64",
	}
	return sh.RunWith(env, "go", "build", "-o", filepath.Join(BuildDir, ProjectName+".exe"), ".")
}

// BuildMac builds the Ebiten game for macOS
func BuildMac() error {
	fmt.Println("Building Ebiten game for macOS...")
	if err := os.MkdirAll(BuildDir, 0755); err != nil {
		return err
	}
	env := map[string]string{
		"GOOS":   "darwin",
		"GOARCH": "amd64",
	}
	return sh.RunWith(env, "go", "build", "-o", filepath.Join(BuildDir, ProjectName), ".")
}

// BuildLinux builds the Ebiten game for Linux
func BuildLinux() error {
	fmt.Println("Building Ebiten game for Linux...")
	if err := os.MkdirAll(BuildDir, 0755); err != nil {
		return err
	}
	env := map[string]string{
		"GOOS":   "linux",
		"GOARCH": "amd64",
	}
	return sh.RunWith(env, "go", "build", "-o", filepath.Join(BuildDir, ProjectName), ".")
}

// BuildWeb builds the Ebiten game for web (WASM)
func BuildWeb() error {
	fmt.Println("Building Ebiten game for web...")
	if err := os.MkdirAll(WebBuildDir, 0755); err != nil {
		return err
	}

	// Ensure public assets exist
	if err := ensureWebAssets(); err != nil {
		return err
	}

	// Copy public folder contents to build/web
	if err := copyDir(PublicDir, WebBuildDir); err != nil {
		return fmt.Errorf("failed to copy public folder: %w", err)
	}

	env := map[string]string{
		"GOOS":   "js",
		"GOARCH": "wasm",
	}
	return sh.RunWith(env, "go", "build", "-o", filepath.Join(WebBuildDir, "game.wasm"), ".")
}

// Clean removes build artifacts
func Clean() error {
	fmt.Println("Cleaning build artifacts...")
	return sh.Rm(BuildDir)
}

// Run builds and runs the Ebiten game
func Run() error {
	mg.Deps(Build)
	fmt.Println("Running Ebiten game...")
	return sh.RunV(filepath.Join(BuildDir, getBinaryName()))
}

// Test runs all tests
func Test() error {
	fmt.Println("Running tests...")
	return sh.Run("go", "test", "./...")
}

// Serve builds web version and starts local development server on port 8080
func Serve() error {
	mg.Deps(BuildWeb)
	return startDevServer(WebBuildDir, 8080)
}

// ServeWithPort builds web version and starts server on specified port
func ServeWithPort(port int) error {
	mg.Deps(BuildWeb)
	return startDevServer(WebBuildDir, port)
}

func getBinaryName() string {
	name := ProjectName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// copyDir copies a directory tree from src to dst
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// startDevServer starts a local HTTP server for web development
func startDevServer(dir string, port int) error {
	addr := "localhost:" + strconv.Itoa(port)

	// Custom file server with proper MIME types for WASM
	fs := http.FileServer(http.Dir(dir))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set appropriate MIME types for WASM files
		if filepath.Ext(r.URL.Path) == ".wasm" {
			w.Header().Set("Content-Type", "application/wasm")
		}

		// Set Cross-Origin-Embedder-Policy and Cross-Origin-Opener-Policy headers
		// These are required for SharedArrayBuffer which may be used by WASM
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")

		fs.ServeHTTP(w, r)
	})

	fmt.Printf("Starting development server at http://%s\n", addr)
	fmt.Printf("Serving files from: %s\n", dir)
	fmt.Println("Press Ctrl+C to stop the server")

	return http.ListenAndServe(addr, handler)
}

// ensureWebAssets checks for public folder and required files, creating them if missing
func ensureWebAssets() error {
	if err := os.MkdirAll(PublicDir, 0755); err != nil {
		return err
	}

	// Check/Create wasm_exec.js
	wasmExecPath := filepath.Join(PublicDir, "wasm_exec.js")
	if _, err := os.Stat(wasmExecPath); os.IsNotExist(err) {
		fmt.Println("wasm_exec.js not found, attempting to copy from GOROOT...")
		goRoot := os.Getenv("GOROOT")
		if goRoot == "" {
			// Try to get GOROOT from go command
			out, err := exec.Command("go", "env", "GOROOT").Output()
			if err == nil {
				goRoot = string(out)
				// Trim newline
				if len(goRoot) > 0 && goRoot[len(goRoot)-1] == '\n' {
					goRoot = goRoot[:len(goRoot)-1]
				}
				// Trim carriage return if on windows
				if len(goRoot) > 0 && goRoot[len(goRoot)-1] == '\r' {
					goRoot = goRoot[:len(goRoot)-1]
				}
			}
		}

		if goRoot != "" {
			src := filepath.Join(goRoot, "misc", "wasm", "wasm_exec.js")
			if err := copyFile(src, wasmExecPath); err != nil {
				fmt.Printf("Warning: Failed to copy wasm_exec.js: %v. You may need to copy it manually.\n", err)
			} else {
				fmt.Println("Copied wasm_exec.js")
			}
		} else {
			fmt.Println("Warning: GOROOT not found. Please copy wasm_exec.js manually to public/wasm_exec.js")
		}
	}

	// Check/Create index.html
	indexPath := filepath.Join(PublicDir, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		fmt.Println("Creating default index.html...")
		html := `<!DOCTYPE html>
<script src="wasm_exec.js"></script>
<script>
// Polyfill
if (!WebAssembly.instantiateStreaming) {
	WebAssembly.instantiateStreaming = async (resp, importObject) => {
		const source = await (await resp).arrayBuffer();
		return await WebAssembly.instantiate(source, importObject);
	};
}

const go = new Go();
WebAssembly.instantiateStreaming(fetch("game.wasm"), go.importObject).then(result => {
	go.run(result.instance);
});
</script>
`
		if err := os.WriteFile(indexPath, []byte(html), 0644); err != nil {
			return err
		}
	}

	return nil
}
