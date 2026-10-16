// Command android prepares the Go library and builds the standard Gradle app.
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"debug/elf"
	"encoding/json"
	"flag"
	"fmt"
	"image/png"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/olivierh59500/passingshot/internal/resources"
)

const ebitenVersion = "v2.10.2"
const applicationID = "com.olivierh.passingshot"

type libraryConfig struct{ Target, Scene, Source string }

func main() {
	target := flag.String("target", "android/arm64", "ebitenmobile target architectures")
	out := flag.String("out", "", "optional copy of the verified debug APK")
	aarOnly := flag.Bool("aar-only", false, "prepare the AAR and original launcher artwork without running Gradle")
	reuse := flag.Bool("reuse-aar", false, "reuse an AAR with matching source and build metadata")
	install := flag.Bool("install", false, "install and launch the debug APK on the selected device")
	serial := flag.String("serial", os.Getenv("ANDROID_SERIAL"), "ADB serial (required when several devices are connected)")
	lockscreen := flag.Bool("probe-lockscreen", false, "allow the debug game window above the keyguard for a device probe")
	scene := flag.String("probe-result", "", "development startup scene: advance, champion or gameover")
	flag.Parse()
	if *scene != "" && *scene != "advance" && *scene != "champion" && *scene != "gameover" {
		check(fmt.Errorf("-probe-result must be advance, champion or gameover"))
	}
	root, err := os.Getwd()
	check(err)
	if _, err := os.Stat("go.mod"); err != nil {
		check(fmt.Errorf("run this command from the repository root"))
	}
	check(os.Setenv("GOCACHE", filepath.Join(root, ".cache", "go-build")))
	home, err := os.UserHomeDir()
	check(err)
	sdk := findDir("Android SDK", os.Getenv("ANDROID_HOME"), os.Getenv("ANDROID_SDK_ROOT"), "/opt/homebrew/share/android-commandlinetools", filepath.Join(home, "Library", "Android", "sdk"), filepath.Join(home, "Android", "Sdk"))
	ndk := os.Getenv("ANDROID_NDK_HOME")
	if ndk == "" {
		ndk = latest(filepath.Join(sdk, "ndk"))
	}
	java := findDir("Java JDK", os.Getenv("JAVA_HOME"), "/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home", "/Applications/Android Studio.app/Contents/jbr/Contents/Home")
	tools := latest(filepath.Join(sdk, "build-tools"))
	if _, err := os.Stat(filepath.Join(sdk, "platforms", "android-36", "android.jar")); err != nil {
		check(fmt.Errorf("Android SDK Platform 36 is required: %w", err))
	}
	for name, value := range map[string]string{"JAVA_HOME": java, "ANDROID_HOME": sdk, "ANDROID_SDK_ROOT": sdk, "ANDROID_NDK_HOME": ndk} {
		check(os.Setenv(name, value))
	}
	check(os.Setenv("PATH", filepath.Join(java, "bin")+string(os.PathListSeparator)+os.Getenv("PATH")))
	// Both flags are needed for LOAD and RELRO boundaries on 16 KiB devices.
	check(os.Setenv("CGO_LDFLAGS", strings.TrimSpace(os.Getenv("CGO_LDFLAGS")+" -Wl,-z,max-page-size=16384 -Wl,-z,common-page-size=16384")))
	aar := filepath.Join(root, "android", "app", "libs", "passingshot.aar")
	check(os.MkdirAll(filepath.Dir(aar), 0755))
	config := libraryConfig{Target: *target, Scene: *scene, Source: sourceDigest()}
	metadata, err := json.Marshal(config)
	check(err)
	if *reuse {
		previous, err := os.ReadFile(aar + ".config")
		check(err)
		if !bytes.Equal(previous, metadata) {
			check(fmt.Errorf("cached AAR source or build options differ; rebuild without -reuse-aar"))
		}
	} else {
		if err := os.Remove(aar + ".config"); err != nil && !os.IsNotExist(err) {
			check(err)
		}
		flags := "-s -w"
		if *scene != "" {
			flags += " -X github.com/olivierh59500/passingshot/mobile.probeResult=" + *scene
		}
		fmt.Println("Building the Go/Ebitengine Android library...")
		run("go", "run", "github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@"+ebitenVersion, "bind", "-target", *target, "-androidapi", "23", "-javapkg", applicationID, "-trimpath", "-ldflags="+flags, "-o", aar, "./mobile")
		check(os.WriteFile(aar+".config", metadata, 0644))
	}
	verifyLibrary(aar)
	r, err := resources.Load()
	check(err)
	iconDir := filepath.Join(root, "android", "app", "build", "generated", "gameResources", "drawable-nodpi")
	check(os.MkdirAll(iconDir, 0755))
	f, err := os.Create(filepath.Join(iconDir, "title.png"))
	check(err)
	check(png.Encode(f, r.Title))
	check(f.Close())
	if *aarOnly {
		fmt.Println("Prepared android/app/libs/passingshot.aar and the original launcher artwork.")
		return
	}
	// Reuse the existing private debug key so installed development builds update.
	key := filepath.Join(root, ".cache", "android-debug.keystore")
	if _, err := os.Stat(key); os.IsNotExist(err) {
		run(filepath.Join(java, "bin", "keytool"), "-genkeypair", "-keystore", key, "-storepass", "android", "-keypass", "android", "-alias", "androiddebugkey", "-dname", "CN=Android Debug,O=Android,C=US", "-keyalg", "RSA", "-keysize", "2048", "-validity", "10000")
	}
	args := []string{"-p", filepath.Join(root, "android"), "--no-daemon", "--console=plain", "-PdebugStoreFile=" + key}
	if *lockscreen {
		args = append(args, "-PprobeLockscreen=true")
	}
	args = append(args, "assembleDebug")
	run(filepath.Join(root, "android", "gradlew"), args...)
	apk := filepath.Join(root, "android", "app", "build", "outputs", "apk", "debug", "app-debug.apk")
	run(filepath.Join(tools, "apksigner"), "verify", "--verbose", apk)
	run(filepath.Join(tools, "zipalign"), "-c", "-P", "16", "4", apk)
	verifyAPK(apk)
	data, err := os.ReadFile(apk)
	check(err)
	if *out != "" {
		check(os.MkdirAll(filepath.Dir(*out), 0755))
		check(os.WriteFile(*out, data, 0644))
		apk = *out
	}
	check(os.WriteFile(apk+".sha256", []byte(fmt.Sprintf("%x  %s\n", sha256.Sum256(data), filepath.Base(apk))), 0644))
	fmt.Printf("Built and verified %s (debug signed).\n", apk)
	if *install {
		adb := filepath.Join(sdk, "platform-tools", "adb")
		if *serial == "" {
			output, err := exec.Command(adb, "devices").Output()
			check(err)
			var devices []string
			for _, line := range strings.Split(string(output), "\n") {
				fields := strings.Fields(line)
				if len(fields) == 2 && fields[1] == "device" {
					devices = append(devices, fields[0])
				}
			}
			if len(devices) != 1 {
				check(fmt.Errorf("select one authorized Android device with -serial or ANDROID_SERIAL (found %d)", len(devices)))
			}
			*serial = devices[0]
		}
		run(adb, "-s", *serial, "install", "-r", apk)
		run(adb, "-s", *serial, "shell", "am", "force-stop", applicationID)
		run(adb, "-s", *serial, "shell", "am", "start", "-n", applicationID+"/.MainActivity")
	}
}

// sourceDigest covers the library's source and embedded data, not local output.
func sourceDigest() string {
	paths := []string{"go.mod", "go.sum"}
	for _, dir := range []string{"internal", "mobile"} {
		check(filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && (strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") || path == "internal/resources/generated/resources.zip" || path == "internal/resources/checksums.json") {
				paths = append(paths, path)
			}
			return nil
		}))
	}
	sort.Strings(paths)
	hash := sha256.New()
	fmt.Fprintln(hash, ebitenVersion)
	for _, path := range paths {
		data, err := os.ReadFile(path)
		check(err)
		fmt.Fprintf(hash, "%s\x00%d\x00", filepath.ToSlash(path), len(data))
		hash.Write(data)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func verifyLibrary(path string) {
	z, err := zip.OpenReader(path)
	check(err)
	defer z.Close()
	count := 0
	for _, f := range z.File {
		if strings.HasPrefix(f.Name, "jni/") && strings.HasSuffix(f.Name, ".so") {
			checkELF(f.Name, readZip(f))
			count++
		}
	}
	if count == 0 {
		check(fmt.Errorf("AAR contains no native libraries"))
	}
}

func verifyAPK(path string) {
	z, err := zip.OpenReader(path)
	check(err)
	defer z.Close()
	count := 0
	for _, f := range z.File {
		if strings.HasPrefix(f.Name, "lib/") && strings.HasSuffix(f.Name, ".so") {
			if f.Method != zip.Store {
				check(fmt.Errorf("native library must be uncompressed: %s", f.Name))
			}
			checkELF(f.Name, readZip(f))
			count++
		}
	}
	if count == 0 {
		check(fmt.Errorf("APK contains no native libraries"))
	}
}

func checkELF(name string, data []byte) {
	f, err := elf.NewFile(bytes.NewReader(data))
	check(err)
	defer f.Close()
	for _, p := range f.Progs {
		if p.Type == elf.PT_LOAD && (p.Align < 16384 || p.Off%16384 != p.Vaddr%16384) {
			check(fmt.Errorf("%s has a load segment without 16 KiB alignment", name))
		}
		if p.Type == elf.PT_GNU_RELRO && (p.Vaddr+p.Memsz)%16384 != 0 {
			check(fmt.Errorf("%s has a RELRO boundary without 16 KiB alignment", name))
		}
	}
}
func readZip(f *zip.File) []byte {
	r, err := f.Open()
	check(err)
	data, err := io.ReadAll(r)
	check(err)
	check(r.Close())
	return data
}
func run(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	check(cmd.Run())
}
func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func findDir(label string, paths ...string) string {
	for _, path := range paths {
		if info, err := os.Stat(path); path != "" && err == nil && info.IsDir() {
			return path
		}
	}
	check(fmt.Errorf("%s not found; configure its environment variable", label))
	return ""
}
func latest(dir string) string {
	entries, err := os.ReadDir(dir)
	check(err)
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		check(fmt.Errorf("no installed tools in %s", dir))
	}
	sort.Strings(names)
	return filepath.Join(dir, names[len(names)-1])
}
