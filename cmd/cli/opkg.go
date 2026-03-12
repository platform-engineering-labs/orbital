package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"charm.land/lipgloss/v2/table"
	"github.com/goforj/godump"
	"github.com/platform-engineering-labs/orbital"
	"github.com/platform-engineering-labs/orbital/platform"
	"github.com/platform-engineering-labs/orbital/platform/arch"
	"github.com/platform-engineering-labs/orbital/platform/os"
	"github.com/spf13/cobra"
)

func init() {
	Opkg.AddCommand(OpkgBuild)

	OpkgBuild.Flags().String("os", "", "override OS")
	OpkgBuild.Flags().String("arch", "", "override architecture")

	OpkgBuild.Flags().String("target-path", "", "Target path for included file system objects")
	OpkgBuild.Flags().String("work-path", "", "Work path for Opkg creation")
	OpkgBuild.Flags().String("output-path", "", "Output path for Opkg")
	OpkgBuild.Flags().Bool("restrict", false, "Restrict included filesystem objects to those present in Opkgfile")
	OpkgBuild.Flags().Bool("secure", false, "Ensure filesystem objects are super user owned")

	Opkg.AddCommand(OpkgContents)
	Opkg.AddCommand(OpkgDump)
	Opkg.AddCommand(OpkgExtract)
	Opkg.AddCommand(OpkgInfo)
	Opkg.AddCommand(OpkgValidate)
}

var Opkg = &cobra.Command{
	Use:     "opkg",
	Short:   "work with Opkgs",
	GroupID: "treepkg",
}

var OpkgBuild = &cobra.Command{
	Use:   "build [OpkgFile Path]",
	Short: "build an Opkg",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		fos, _ := cmd.Flags().GetString("os")
		farch, _ := cmd.Flags().GetString("arch")

		targetPath, _ := cmd.Flags().GetString("target-path")
		outputPath, _ := cmd.Flags().GetString("output-path")
		workPath, _ := cmd.Flags().GetString("work-path")
		restrict, _ := cmd.Flags().GetBool("restrict")
		secure, _ := cmd.Flags().GetBool("secure")

		var pltfrm *platform.Platform
		if fos != "" && farch != "" {
			pltfrm = &platform.Platform{OS: os.OS(fos), Arch: arch.Arch(farch)}
		}

		orb, err := orbital.Dynamic(cfgPath, slog.New(Logger))
		if err != nil {
			return err
		}

		_, pkgPath, err := orb.Opkg.Build(cmd.Flags().Arg(0), pltfrm, targetPath, workPath, outputPath, restrict, secure)
		if err != nil {
			return err
		}

		fmt.Println(fmt.Sprintf("built opkg: %s", pkgPath))

		return nil
	},
}

var OpkgDump = &cobra.Command{
	Use:   "dump [Opkg Path]",
	Short: "dump Opkg manifest",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		if cmd.Flags().Arg(0) == "" {
			return errors.New("missing opkg path")
		}

		orb, err := orbital.Dynamic(cfgPath, slog.New(Logger))
		if err != nil {
			return err
		}

		manifest, err := orb.Opkg.Manifest(cmd.Flags().Arg(0))
		if err != nil {
			return err
		}

		godump.Dump(manifest)

		return nil
	},
}

var OpkgExtract = &cobra.Command{
	Use:   "extract [Opkg Path]",
	Short: "extract the contents of an Opkg",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		if cmd.Flags().Arg(0) == "" {
			return errors.New("missing opkg path")
		}

		targetPath := "./"
		if cmd.Flags().Arg(1) != "" {
			targetPath = cmd.Flags().Arg(1)
		}

		orb, err := orbital.Dynamic(cfgPath, slog.New(Logger))
		if err != nil {
			return err
		}

		return orb.Opkg.Extract(cmd.Flags().Arg(0), targetPath)
	},
}

var OpkgContents = &cobra.Command{
	Use:   "contents [Opkg Path]",
	Short: "list Opkg contents",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		if cmd.Flags().Arg(0) == "" {
			return errors.New("missing opkg path")
		}

		orb, err := orbital.Dynamic(cfgPath, slog.New(Logger))
		if err != nil {
			return err
		}

		manifest, err := orb.Opkg.Manifest(cmd.Flags().Arg(0))
		if err != nil {
			return err
		}

		for _, f := range manifest.Contents() {
			fmt.Println(strings.Join(f.Columns(), "\t"))
		}

		return nil
	},
}

var OpkgInfo = &cobra.Command{
	Use:   "info [Opkg Path]",
	Short: "show Opkg metadata",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		if cmd.Flags().Arg(0) == "" {
			return errors.New("missing opkg path")
		}

		orb, err := orbital.Dynamic(cfgPath, slog.New(Logger))
		if err != nil {
			return err
		}

		manifest, err := orb.Opkg.Manifest(cmd.Flags().Arg(0))
		if err != nil {
			return err
		}

		tbl := table.New()
		tbl.Headers("Package")

		tbl.Row("Name", manifest.Name)
		tbl.Row("Version", manifest.Version.String())
		tbl.Row("Publisher", manifest.Publisher)
		tbl.Row("Platform", manifest.Platform().String())
		tbl.Row("Summary", manifest.Summary)
		tbl.Row("Description", manifest.Description)

		fmt.Println(tbl.Render())

		tbl = table.New()
		tbl.Headers("Metadata")

		for ns, meta := range manifest.Metadata {
			for k, v := range meta {
				tbl.Row(fmt.Sprintf("%s:%s", ns, k), v)
			}
		}

		fmt.Println(tbl.Render())

		return nil
	},
}

var OpkgValidate = &cobra.Command{
	Use:   "validate [Opkg Path]",
	Short: "validate Opkg signature and contents",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		if cmd.Flags().Arg(0) == "" {
			return errors.New("missing opkg path")
		}

		orb, err := orbital.Dynamic(cfgPath, slog.New(Logger))
		if err != nil {
			return err
		}

		err = orb.Opkg.Validate(cmd.Flags().Arg(0))
		if err != nil {
			return err
		}

		fmt.Println("OK")

		return nil
	},
}
