package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GddonObject represents the main configuration structure
type GddonObject struct {
	Packages []GddonPackage `json:"packages"`
}

// GddonPackage represents a package in the configuration
type GddonPackage struct {
	Name    string `json:"name"`
	GitRepo string `json:"git_repo"`
	Commit  string `json:"commit"`
	Links   []Link `json:"links"`
}

// Link represents a file link between source and target
type Link struct {
	TargetFolder string `json:"target_folder"`
	SourceFolder string `json:"source_folder"`
}

// logError prints an error message
func logError(message string) {
	fmt.Printf("\033[31mError: %s\033[0m\n", message)
}

// logInfo prints an info message
func logInfo(message string) {
	fmt.Printf("\033[36mInfo: %s\033[0m\n", message)
}

// logCheck prints a success message
func logCheck(message string) {
	fmt.Printf("\033[32m✓ %s\033[0m\n", message)
}

// runShellCommand executes a shell command
func runShellCommand(command string, workingDir string, verbose bool) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = workingDir

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// assertResult checks if an error exists and logs it
func assertResult(err error, message string) {
	if err != nil {
		logError(message)
		os.Exit(1)
	}
}

// SearchProjectRoot finds the root directory of the project
func SearchProjectRoot() string {
	path, err := filepath.Abs(".")
	if err != nil {
		logError("Could not get absolute path")
		os.Exit(1)
	}

	for {
		godotProject := filepath.Join(path, "project.godot")
		if _, err := os.Stat(godotProject); err == nil {
			break
		}

		parent := filepath.Dir(path)
		if parent == path {
			logError("Godot project not found!")
			os.Exit(1)
		}
		path = parent
	}

	logCheck(fmt.Sprintf("Found root project in: %s", path))
	return path
}

// Initialize sets up the basic project structure
func Initialize(root string) {
	gitIgnore := filepath.Join(root, ".gitignore")
	if _, err := os.Stat(gitIgnore); os.IsNotExist(err) {
		err := os.WriteFile(gitIgnore, []byte(createGitIgnoreFile()), 0644)
		if err != nil {
			logError("There was a problem creating the .gitignore file!")
			os.Exit(1)
		}
	}
}

// InitializeGddonFiles creates necessary GDDON configuration files
func InitializeGddonFiles(root string) {
	// Create .gddon.d/ folder if it doesn't exist
	gddonDir := filepath.Join(root, ".gddon.d")
	if _, err := os.Stat(gddonDir); os.IsNotExist(err) {
		err := os.MkdirAll(gddonDir, 0755)
		assertResult(err, "Couldn't create .gddon.d/ folder!")

		gdIgnore := filepath.Join(gddonDir, ".gdignore")
		if _, err := os.Stat(gdIgnore); os.IsNotExist(err) {
			err := os.WriteFile(gdIgnore, []byte(createGdIgnoreFile()), 0644)
			if err != nil {
				logError("There was a problem creating the .gdignore file!")
				os.Exit(1)
			}
		}

		logInfo("Created .gddon.d/ folder")
	}

	// Create ,gddon file if it doesn't exist
	gddonFile := filepath.Join(root, ".gddon")
	if _, err := os.Stat(gddonFile); os.IsNotExist(err) {
		err := os.WriteFile(gddonFile, []byte(createGddonFile()), 0644)
		assertResult(err, "Couldn't create ,gddon file!")
		logInfo("Created ,gddon file")
	}
}

// CheckInitialization verifies if all required files exist
func CheckInitialization(root string) bool {
	gitIgnore := filepath.Join(root, ".gitignore")
	if _, err := os.Stat(gitIgnore); os.IsNotExist(err) {
		logError(".gitignore file does not exist!")
	}

	ret := true
	gddonFile := filepath.Join(root, ".gddon")
	if _, err := os.Stat(gddonFile); os.IsNotExist(err) {
		logError(".gddon file does not exist!")
		ret = false
	}

	gddonDir := filepath.Join(root, ".gddon.d")
	if _, err := os.Stat(gddonDir); os.IsNotExist(err) {
		logError(".gddon.d/ folder does not exist!")
		ret = false
	}
	return ret
}

// Helper functions for creating configuration files
func createGitIgnoreFile() string {
	return `# Godot-specific ignores
*.translation
export_presets.cfg
.godot/
.gddon.d/
`
}

func createGdIgnoreFile() string {
	return `# Ignore everything in this directory
*
`
}

func createGddonFile() string {
	return `{
  "packages": []
}`
}

// InstallRepositories installs all packages defined in the ,gddon file
func InstallRepositories(root string, verbose bool) {
	gddonFilePath := filepath.Join(root, ".gddon")
	gddonObject := readGddonFile(gddonFilePath)

	for i := range gddonObject.Packages {
		pkg := &gddonObject.Packages[i]
		logInfo(fmt.Sprintf("Installing %s...", pkg.Name))
		cloneOrFetchPackage(root, pkg, verbose)
		installGddonPackage(root, pkg.Commit, pkg, false, true, verbose)
	}

	writeGddonFile(gddonFilePath, &gddonObject)
}

// AddRepository adds a new repository to the project
func AddRepository(root string, gitRepo string, verbose bool) {
	gddonFilePath := filepath.Join(root, ".gddon")
	gddonObject := readGddonFile(gddonFilePath)

	if findPackageByRepository(gddonObject.Packages, gitRepo) != -1 {
		logError("Repository already exists!")
		os.Exit(1)
	}

	defaultName := getRepoName(gitRepo)
	name := promptText("Name of the addon:", defaultName)

	if findPackageByName(gddonObject.Packages, name) != -1 {
		logError("Addon name exists!")
		os.Exit(1)
	}

	commit := promptText("Commit hash of the repository:", "latest")

	newPackage := GddonPackage{
		Name:    name,
		GitRepo: gitRepo,
		Commit:  commit,
		Links:   []Link{},
	}

	gddonObject.Packages = append(gddonObject.Packages, newPackage)
	targetPackage := &gddonObject.Packages[len(gddonObject.Packages)-1]

	cloneOrFetchPackage(root, targetPackage, verbose)
	installGddonPackage(root, commit, targetPackage, false, true, verbose)

	writeGddonFile(gddonFilePath, &gddonObject)
}

// UpdateRepository updates a specific repository
func UpdateRepository(root string, verbose bool) {
	gddonFilePath := filepath.Join(root, ".gddon")
	gddonObject := readGddonFile(gddonFilePath)

	if len(gddonObject.Packages) == 0 {
		logError("No addons to update!")
		os.Exit(1)
	}

	options := make([]string, len(gddonObject.Packages))
	for i, pkg := range gddonObject.Packages {
		options[i] = pkg.Name
	}

	ans := promptSelect("Which addon you want to update?", options)
	packageIndex := findPackageByName(gddonObject.Packages, ans)
	targetPackage := &gddonObject.Packages[packageIndex]

	logInfo(fmt.Sprintf("Updating %s...", targetPackage.Name))
	cloneOrFetchPackage(root, targetPackage, verbose)
	installGddonPackage(root, "", targetPackage, true, true, verbose)

	writeGddonFile(gddonFilePath, &gddonObject)
}

// Helper functions for package management
func readGddonFile(filePath string) GddonObject {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		err := os.WriteFile(filePath, []byte(createGddonFile()), 0644)
		assertResult(err, "Couldn't create ,gddon file!")
	}

	content, err := os.ReadFile(filePath)
	assertResult(err, "Couldn't read ,gddon file!")

	var gddonObject GddonObject
	err = json.Unmarshal(content, &gddonObject)
	assertResult(err, "Couldn't parse ,gddon file!")

	return gddonObject
}

func writeGddonFile(filePath string, gddonObject *GddonObject) {
	jsonData, err := json.MarshalIndent(gddonObject, "", "  ")
	assertResult(err, "Couldn't marshal ,gddon file!")

	err = os.WriteFile(filePath, jsonData, 0644)
	assertResult(err, "Couldn't write ,gddon file!")
}

func findPackageByName(packages []GddonPackage, name string) int {
	for i, pkg := range packages {
		if pkg.Name == name {
			return i
		}
	}
	return -1
}

func findPackageByRepository(packages []GddonPackage, repo string) int {
	for i, pkg := range packages {
		if pkg.GitRepo == repo {
			return i
		}
	}
	return -1
}

func getRepoName(gitRepo string) string {
	parts := strings.Split(gitRepo, "/")
	return strings.TrimSuffix(parts[len(parts)-1], ".git")
}

// cloneOrFetchPackage clones or fetches updates for a package
func cloneOrFetchPackage(root string, package_ *GddonPackage, verbose bool) {
	packagePath := filepath.Join(root, ".gddon.d", package_.Name)

	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		cmd := fmt.Sprintf("cd .gddon.d/ && git clone %s %s --progress", package_.GitRepo, package_.Name)
		err := runShellCommand(cmd, root, verbose)
		assertResult(err, "Couldn't clone repository!")
		logCheck("Created package folder on .gddon.d")
	} else {
		if package_.GitRepo == "" {
			cmd := fmt.Sprintf("cd .gddon.d/%s && git remote get-url origin", package_.Name)
			out, err := exec.Command("sh", "-c", cmd).Output()
			if err != nil {
				logError("GDDON Package has no origin yet!")
				os.Exit(1)
			}
			package_.GitRepo = strings.TrimSpace(string(out))
		}

		cmd := fmt.Sprintf("cd .gddon.d/%s && git fetch origin && git pull", package_.Name)
		err := runShellCommand(cmd, root, verbose)
		assertResult(err, "Couldn't fetch package repository updates!")
		logInfo("Glam package folder already exists, fetched and pulled latest changes")
	}
}

// installGddonPackage installs a package to the project
func installGddonPackage(root string, commit string, package_ *GddonPackage, updatePackage bool, copyFiles bool, verbose bool) {
	if updatePackage {
		package_.Commit = "latest"
	}

	if commit != "latest" {
		package_.Commit = commit
	}

	// Get all folders in addon
	folders := listDir(fmt.Sprintf(".gddon.d/%s/addons", package_.Name))
	if len(package_.Links) == 0 {
		if len(folders) == 1 {
			package_.Links = append(package_.Links, Link{
				TargetFolder: fmt.Sprintf("addons/%s", folders[0]),
				SourceFolder: fmt.Sprintf("addons/%s", folders[0]),
			})
		} else {
			// TODO: Implement multi-select prompt
			logError("Multiple addons not yet supported")
			os.Exit(1)
		}
	}

	if package_.Commit == "latest" {
		cmd := fmt.Sprintf("cd .gddon.d/%s && git rev-parse HEAD", package_.Name)
		out, err := exec.Command("sh", "-c", cmd).Output()
		assertResult(err, "Couldn't get latest commit!")
		package_.Commit = strings.TrimSpace(string(out))
	} else {
		logInfo("Git checkout to package commit")
		cmd := fmt.Sprintf("cd .gddon.d/%s && git reset --hard %s", package_.Name, package_.Commit)
		err := runShellCommand(cmd, root, verbose)
		assertResult(err, "Couldn't checkout repository!")
	}

	if copyFiles {
		for _, link := range package_.Links {
			// Create target directory if it doesn't exist
			targetPath := filepath.Join(root, link.TargetFolder)
			err := os.MkdirAll(targetPath, 0755)
			assertResult(err, "Couldn't create addons folder!")

			// Copy files
			sourcePath := filepath.Join(root, ".gddon.d", package_.Name, link.SourceFolder)
			// cmd := fmt.Sprintf("cp -rf %s/* %s", sourcePath, targetPath)
			// err = runShellCommand(cmd, root, verbose)
			err = copyDir(targetPath, sourcePath)
			assertResult(err, "Couldn't copy files to addons!")
		}
	}
}

// promptText prompts the user for text input
func promptText(prompt string, defaultValue string) string {
	fmt.Printf("%s [%s]: ", prompt, defaultValue)
	var input string
	fmt.Scanln(&input)
	if input == "" {
		return defaultValue
	}
	return input
}

// promptSelect prompts the user to select from a list of options
func promptSelect(prompt string, options []string) string {
	fmt.Println(prompt)
	for i, option := range options {
		fmt.Printf("%d. %s\n", i+1, option)
	}

	var choice int
	for {
		fmt.Print("Enter your choice (1-", len(options), "): ")
		_, err := fmt.Scan(&choice)
		if err == nil && choice >= 1 && choice <= len(options) {
			return options[choice-1]
		}
		fmt.Println("Invalid choice. Please try again.")
	}
}

// CreateAddon creates a new addon package
func CreateAddon(root string, verbose bool) {
	gddonFilePath := filepath.Join(root, ".gddon")
	gddonObject := readGddonFile(gddonFilePath)

	folders := listAddons(root, verbose)
	if len(folders) == 0 {
		logError("No addons found in the project!")
		os.Exit(1)
	}

	addonName := promptSelect("Which addon you'll create a repository?", folders)

	if findPackageByLink(gddonObject.Packages, addonName) != -1 {
		logError("There is a repository linked to that addon already!")
		os.Exit(1)
	}

	repoName := promptText("Name of the repository:", addonName)

	// Create repository structure
	repoPath := filepath.Join(root, ".gddon.d", repoName, "addons", addonName)
	err := os.MkdirAll(repoPath, 0755)
	assertResult(err, "Repository folder failed to be created!")

	// Initialize git repository
	cmd := fmt.Sprintf("cd .gddon.d/%s && git init", repoName)
	err = runShellCommand(cmd, root, verbose)
	assertResult(err, "Repository failed to be initialized!")

	// Add package to configuration
	gddonObject.Packages = append(gddonObject.Packages, GddonPackage{
		Name:    repoName,
		GitRepo: "",
		Commit:  "",
		Links: []Link{
			{
				TargetFolder: fmt.Sprintf("addons/%s", addonName),
				SourceFolder: fmt.Sprintf("addons/%s", addonName),
			},
		},
	})

	writeGddonFile(gddonFilePath, &gddonObject)

	targetPackage := &gddonObject.Packages[len(gddonObject.Packages)-1]
	applyPackageFiles(root, targetPackage, verbose)
}

// ApplyChanges applies changes from the project to a selected package
func ApplyChanges(root string, verbose bool) {
	gddonFilePath := filepath.Join(root, ".gddon")
	gddonObject := readGddonFile(gddonFilePath)

	if len(gddonObject.Packages) == 0 {
		logError("No addons to apply changes!")
		os.Exit(1)
	}

	options := make([]string, len(gddonObject.Packages))
	for i, pkg := range gddonObject.Packages {
		options[i] = pkg.Name
	}

	ans := promptSelect("Which addon you want to apply changes?", options)
	packageIndex := findPackageByName(gddonObject.Packages, ans)
	targetPackage := &gddonObject.Packages[packageIndex]

	applyPackageFiles(root, targetPackage, verbose)
	writeGddonFile(gddonFilePath, &gddonObject)
}

// listAddons returns a list of addon folders in the project
func listAddons(_ string, _ bool) []string {
	return listDir("addons")
}

// findPackageByLink finds a package by its link target
func findPackageByLink(packages []GddonPackage, addonsFolder string) int {
	for i, pkg := range packages {
		for _, link := range pkg.Links {
			if link.TargetFolder == fmt.Sprintf("addons/%s", addonsFolder) {
				return i
			}
		}
	}
	return -1
}

// applyPackageFiles applies changes from the project to the package
func applyPackageFiles(root string, package_ *GddonPackage, verbose bool) {
	for _, link := range package_.Links {
		// Overwrite source folder with target folder
		sourcePath := filepath.Join(root, ".gddon.d", package_.Name, link.SourceFolder)
		targetPath := filepath.Join(root, link.TargetFolder)

		// Remove existing files in source
		// cmd := fmt.Sprintf("rm -rf %s/*", sourcePath)
		// err := runShellCommand(cmd, root, verbose)
		err := removeContents(sourcePath)
		assertResult(err, "Couldn't overwrite source folder files!")

		// Copy files from target to source
		// cmd = fmt.Sprintf("cp -rf %s/* %s", targetPath, sourcePath)
		// err = runShellCommand(cmd, root, verbose)
		err = copyDir(targetPath, sourcePath)
		assertResult(err, "Couldn't copy files to repository!")
	}
}

func listDir(src string) []string {
	res := []string{}
	entries, err := os.ReadDir(src)
	if err != nil {
		logError(err.Error())
		return res
	}
	for _, entry := range entries {
		res = append(res, entry.Name())
	}
	return res
}

// copyDir recursively copies a directory tree
func copyDir(src, dst string) error {
	// Get properties of source dir
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination dir
	err = os.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = copyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			// Copy the file
			err = copyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file
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
	if err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

// removeContents removes all contents of a directory
func removeContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			err = os.RemoveAll(path)
		} else {
			err = os.Remove(path)
		}
		if err != nil {
			return err
		}
	}

	return nil
}
