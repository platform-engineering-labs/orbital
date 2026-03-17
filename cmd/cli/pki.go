package cli

import (
	"fmt"
	"log/slog"

	"github.com/platform-engineering-labs/orbital"
	"github.com/platform-engineering-labs/orbital/opm/pki"
	"github.com/platform-engineering-labs/orbital/x/collections"
	"github.com/spf13/cobra"
)

func init() {
	Pki.AddCommand(PkiKeyPair)
	Pki.AddCommand(PkiTrust)

	PkiKeyPairImport.Flags().StringP("method", "m", "file", "methods: file | env")
	PkiTrustImport.Flags().StringP("method", "m", "file", "methods: file | dns")

	PkiKeyPair.AddCommand(PkiKeyPairImport)
	PkiKeyPair.AddCommand(PkiKeyPairList)
	PkiKeyPair.AddCommand(PkiKeyPairRemove)

	PkiTrust.AddCommand(PkiTrustImport)
	PkiTrust.AddCommand(PkiTrustList)
	PkiTrust.AddCommand(PkiTrustRefresh)
	PkiTrust.AddCommand(PkiTrustRemove)
}

var Pki = &cobra.Command{
	Use:     "pki",
	Short:   "manage pki store",
	GroupID: "manage",
}

var PkiKeyPair = &cobra.Command{
	Use:   "keypair",
	Short: "manage signing keypairs",
}

var PkiTrust = &cobra.Command{
	Use:   "trust",
	Short: "manage trust store",
}

var PkiKeyPairImport = &cobra.Command{
	Use:   "import [certPath] [keyPath] | [certEnvVar] [keyEnvVar]",
	Short: "import keypair",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")
		method, _ := cmd.Flags().GetString("method")

		if method == "file" {
			if cmd.Flags().Arg(0) == "" || cmd.Flags().Arg(1) == "" {
				return fmt.Errorf("must provide certPath and keyPath args")
			}
		} else if method == "env" {
			if cmd.Flags().Arg(0) == "" || cmd.Flags().Arg(1) == "" {
				return fmt.Errorf("must provide certEnvVar and keyEnvVar args")
			}
		} else {
			return fmt.Errorf("unknown method: %s", method)
		}

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		return orb.Pki.KeyPairImport(method, cmd.Flags().Arg(0), cmd.Flags().Arg(1))
	},
}

var PkiKeyPairList = &cobra.Command{
	Use:   "list",
	Short: "list keypairs",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		keypairs, err := orb.Pki.KeyPairList()
		if err != nil {
			return err
		}

		for _, keypair := range keypairs {
			fmt.Println(fmt.Sprintf("%s\t%s\t%s\t%s", keypair.SKI, keypair.Subject, keypair.Publisher, keypair.Fingerprint))
		}

		return nil
	},
}

var PkiKeyPairRemove = &cobra.Command{
	Use:   "remove [SKI]",
	Short: "remove keypair",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		if cmd.Flags().Arg(0) == "" {
			return fmt.Errorf("must provide SKI")
		}

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		return orb.Pki.KeyPairRemove(cmd.Flags().Arg(0))
	},
}

var PkiTrustList = &cobra.Command{
	Use:   "list",
	Short: "list trusted certificates",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		certs, err := orb.Pki.TrustList()
		if err != nil {
			return err
		}

		publishers := collections.Map(certs, func(c *pki.CertEntry) string {
			return c.Publisher
		})

		publishers = collections.Unique(publishers)

		for _, publisher := range publishers {
			fmt.Println(publisher)
			for _, cert := range certs {
				if cert.Publisher == publisher {
					fmt.Println(fmt.Sprintf("  %s\t%s\t%s", cert.SKI, cert.Subject, cert.Fingerprint))
				}
			}
			fmt.Println()
		}

		return nil
	},
}

var PkiTrustImport = &cobra.Command{
	Use:   "import [certPath ...] | [ski] [publisher]",
	Short: "import trusted certificate(s)",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")
		method, _ := cmd.Flags().GetString("method")

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		if method == "file" {
			return orb.Pki.TrustImportFiles(cmd.Flags().Args()...)
		} else {
			if cmd.Flags().NArg() < 2 {
				return fmt.Errorf("must provide ski and publisher")
			}

			return orb.Pki.TrustImportDNS(cmd.Flags().Arg(0), cmd.Flags().Arg(1))
		}
	},
}

var PkiTrustRefresh = &cobra.Command{
	Use:   "refresh",
	Short: "refresh trusted certificates",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		return orb.Pki.TrustRefresh()
	},
}

var PkiTrustRemove = &cobra.Command{
	Use:   "remove [SKI]",
	Short: "remove trusted certificate",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, _ := cmd.Flags().GetString("config")

		if cmd.Flags().Arg(0) == "" {
			return fmt.Errorf("must provide SKI")
		}

		orb, err := orbital.Dynamic(slog.New(Logger), cfgPath)
		if err != nil {
			return err
		}

		return orb.Pki.TrustRemove(cmd.Flags().Arg(0))
	},
}
