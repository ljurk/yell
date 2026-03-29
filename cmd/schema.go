package cmd

import (
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	whisper "github.com/go-graphite/go-whisper"
	"github.com/spf13/cobra"

	"github.com/ljurk/yell/lib"
)

var (
	schema      string
	xff         float32
	compressed  bool
	aggregation string
	rootDir     string
	schemaCmd   = &cobra.Command{
		Use:   "schema",
		Short: "command to run analysis in comparison to a storage-schemas.conf",
	}
	applyCmd = &cobra.Command{
		Use:   "apply [whisper-dir]",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Short: "resize whisper files to match the defined retentions in storage-schemas.conf",
		Run: func(cmd *cobra.Command, args []string) {
			schemaPath, _ := cmd.Flags().GetString("schema")
			schemas, err := lib.ParseStorageSchemas(schemaPath)
			if err != nil {
				log.Fatalf("failed to parse schemas %s: %v\n", schemaPath, err)
			}

			defaultSchema := findDefaultSchema(schemas)
			if defaultSchema == nil {
				log.Fatalf("no default schema found in storage-schemas.conf\n")
			}

			pathArg := args[0]
			rootForMetric := rootDir
			if rootForMetric == "" {
				rootForMetric = pathArg
			}

			if rootDir != "" {
				if !lib.IsDirectory(rootDir) {
					log.Fatalf("root-dir %s does not exist or is not a directory\n", rootDir)
				}
			}

			var files []string
			if lib.IsDirectory(pathArg) {
				files, err = lib.FindWhisperFiles(pathArg)
				if err != nil {
					log.Fatalf("failed walking root %s: %v\n", pathArg, err)
				}
			} else if lib.IsWhisperFile(pathArg) {
				files = []string{pathArg}
			} else {
				log.Fatalf("%s is not a directory or .wsp file\n", pathArg)
			}

			if len(files) == 0 {
				log.Fatalf("no .wsp files found\n")
			}

			wr := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(wr, "status\tmetric\told\tnew\tdetail")

			hasErrors := false
			for _, f := range files {
				metric := lib.MetricFromPath(rootForMetric, f)

				matched := findMatchingSchema(metric, schemas)
				if matched == nil {
					log.Fatalf("no schema matched for metric %s\n", metric)
				}

				oldRet, newRet, resized, err := lib.ResizeWhisperFile(f, matched.Retentions, lib.ResizeOptions{
					XFF:         xff,
					Compressed:  compressed,
					Aggregation: aggregation,
				})

				if err != nil {
					_, _ = fmt.Fprintf(wr, "ERROR\t%s\t-\t-\t%s\n", metric, err)
					hasErrors = true
					continue
				}

				if resized {
					_, _ = fmt.Fprintf(wr, "RESIZED\t%s\t%s\t%s\t-\n", metric, oldRet, newRet)
				} else {
					_, _ = fmt.Fprintf(wr, "SKIP\t%s\t%s\t%s\talready matching\n", metric, oldRet, newRet)
				}
			}
			err = wr.Flush()
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, "ERROR failed to close TabWriter")
			}
			if hasErrors {
				os.Exit(1)
			}
		},
	}
	checkCmd = &cobra.Command{
		Use:   "check [whisper-dir]",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Short: "check if whisper files are matching the defined retentions",
		Run: func(cmd *cobra.Command, args []string) {
			schemaPath, _ := cmd.Flags().GetString("schema")
			schemas, err := lib.ParseStorageSchemas(schemaPath)
			if err != nil {
				log.Fatalf("failed to parse schemas %s: %v\n", schemaPath, err)
			}

			pathArg := args[0]
			rootForMetric := rootDir
			if rootForMetric == "" {
				rootForMetric = pathArg
			}

			if rootDir != "" {
				if !lib.IsDirectory(rootDir) {
					log.Fatalf("root-dir %s does not exist or is not a directory\n", rootDir)
				}
			}

			var files []string
			if lib.IsDirectory(pathArg) {
				files, err = lib.FindWhisperFiles(pathArg)
				if err != nil {
					log.Fatalf("failed walking root %s: %v\n", pathArg, err)
				}
			} else if lib.IsWhisperFile(pathArg) {
				files = []string{pathArg}
			} else {
				log.Fatalf("%s is not a directory or .wsp file\n", pathArg)
			}

			if len(files) == 0 {
				log.Fatalf("no .wsp files found\n")
			}

			wr := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(wr, "status\tmetric\texpected\tactual\tdetail")

			for _, f := range files {
				metric := lib.MetricFromPath(rootForMetric, f)

				var matched *lib.Schema
				for i := range schemas {
					s := &schemas[i]
					if s.Pattern == nil {
						continue
					}
					if s.Pattern.MatchString(metric) {
						matched = s
						break
					}
				}

				if matched == nil {
					_, _ = fmt.Fprintf(wr, "NOMATCH\t%s\t-\t-\tno schema matched\n", metric)
					continue
				}

				var wf *whisper.Whisper
				wf, err = whisper.Open(f)
				if err != nil {
					_, _ = fmt.Fprintf(wr, "ERROR\t%s\t-\t-\tfailed to open: %v\n", metric, err)
					continue
				}
				actualSpecs := lib.WhisperRetentionsToSpecs(wf.Retentions())
				err = wf.Close()
				if err != nil {
					_, _ = fmt.Fprintf(wr, "ERROR\t%s\t-\t-\tfailed to close: %v\n", pathArg, err)
					return
				}

				expectedStr := lib.FormatRetentionList(matched.Retentions)
				actualStr := lib.FormatRetentionList(actualSpecs)
				if lib.CompareSpecsEqual(actualSpecs, matched.Retentions) {
					_, _ = fmt.Fprintf(wr, "OK\t%s\t%s\t%s\tmatched schema[%s]\n", metric, expectedStr, actualStr, matched.Name)
				} else {
					_, _ = fmt.Fprintf(wr, "MISMATCH\t%s\texpected:%s\tgot:%s\tschema[%s]\n", metric, expectedStr, actualStr, matched.Name)
				}
			}
			err = wr.Flush()
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, "ERROR failed to close TabWriter")
			}
		},
	}
	countCmd = &cobra.Command{
		Use:   "count [whisper-dir]",
		Args:  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Short: "count matching metrics per definition",
		Run: func(cmd *cobra.Command, args []string) {
			path, _ := cmd.Flags().GetString("schema")

			//parse storage-schemas
			schemas, err := lib.ParseStorageSchemas(path)
			if err != nil {
				log.Fatalf("failed to parse schemas %s: %v\n", path, err)
			}

			// find all .wsp files under path
			var files []string
			files, err = lib.FindWhisperFiles(args[0])
			if err != nil {
				log.Fatalf("failed walking root %s: %v\n", args[0], err)
			}
			if len(files) == 0 {
				log.Fatalf("no .wsp files found under %s\n", args[0])
			}

			// output table header
			wr := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(wr, "count\tname\tpattern")
			schemaCounts, _ := lib.CountDefinitions(schemas, args[0], files)
			for _, i := range schemaCounts {
				_, _ = fmt.Fprintf(wr, "%d\t%s\t%s\n", i.Count, i.Definition.Name, i.Definition.Pattern)
			}
			err = wr.Flush()
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, "ERROR failed to close TabWriter")
			}
		},
	}
)

func init() {
	schemaCmd.PersistentFlags().StringVarP(&schema, "schema", "s", "", "path to storage-schemas.conf")
	_ = schemaCmd.MarkPersistentFlagRequired("schema")
	schemaCmd.PersistentFlags().StringVar(&rootDir, "root-dir", "", "root directory for metric name calculation")

	applyCmd.Flags().Float32VarP(&xff, "xff", "x", 0.5, "xFilesFactor")
	applyCmd.Flags().BoolVar(&compressed, "compressed", false, "use compressed format (default: keep current)")
	applyCmd.Flags().StringVarP(&aggregation, "aggregate", "a", "average", "aggregation method (average, sum, last, max, min, first)")

	schemaCmd.AddCommand(countCmd)
	schemaCmd.AddCommand(checkCmd)
	schemaCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(schemaCmd)
}

func findDefaultSchema(schemas []lib.Schema) *lib.Schema {
	for i := range schemas {
		s := &schemas[i]
		if s.Pattern == nil {
			continue
		}
		if s.Pattern.MatchString("some.test.metric") {
			return s
		}
	}
	return nil
}

func findMatchingSchema(metric string, schemas []lib.Schema) *lib.Schema {
	for i := range schemas {
		s := &schemas[i]
		if s.Pattern == nil {
			continue
		}
		if s.Pattern.MatchString(metric) {
			return s
		}
	}
	return nil
}
