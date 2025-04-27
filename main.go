package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "gddon",
	Short: "Gddon - Godot Library Addon Manager",
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Godot project for gddon",
	Run: func(cmd *cobra.Command, args []string) {
		root := SearchProjectRoot()
		InitializeGddonFiles(root)
		Initialize(root)
	},
}

var addCmd = &cobra.Command{
	Use:   "add [git_repo]",
	Short: "Add new repository",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := SearchProjectRoot()
		if CheckInitialization(root) {
			AddRepository(root, args[0], verbose)
		}
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a repository from an existing addon",
	Run: func(cmd *cobra.Command, args []string) {
		root := SearchProjectRoot()
		if CheckInitialization(root) {
			CreateAddon(root, verbose)
		}
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a repository",
	Run: func(cmd *cobra.Command, args []string) {
		root := SearchProjectRoot()
		if CheckInitialization(root) {
			UpdateRepository(root, verbose)
		}
	},
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install all addons on gddons file",
	Run: func(cmd *cobra.Command, args []string) {
		root := SearchProjectRoot()
		if CheckInitialization(root) {
			InstallRepositories(root, verbose)
		}
	},
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply changes to a repository",
	Run: func(cmd *cobra.Command, args []string) {
		root := SearchProjectRoot()
		if CheckInitialization(root) {
			ApplyChanges(root, verbose)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose (output subshell commands)")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(applyCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
