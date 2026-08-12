package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"thing-index/internal/indeximport"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Printf("index import failed: %v", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("index-import", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var databaseURL string
	var path string

	defaultDatabaseURL := firstNonEmpty(os.Getenv("THINGINDEX_IMPORT_DATABASE_URL"), os.Getenv("DATABASE_URL"))
	fs.StringVar(&databaseURL, "database-url", defaultDatabaseURL, "PostgreSQL DSN or URL for the external database")
	fs.StringVar(&databaseURL, "database_url", defaultDatabaseURL, "PostgreSQL DSN or URL for the external database")
	fs.StringVar(&path, "path", "", "Path to an exported .json file or a directory with export files")

	if err := fs.Parse(args); err != nil {
		return err
	}

	db, err := indeximport.OpenDatabase(databaseURL)
	if err != nil {
		return err
	}

	service := indeximport.NewService(db)
	report, err := service.ImportPath(context.Background(), path)
	printReport(report)
	if err != nil {
		return err
	}

	return nil
}

func printReport(report *indeximport.ImportReport) {
	if report == nil {
		return
	}

	fmt.Fprintf(os.Stdout, "discovered_files=%d created=%d updated=%d skipped=%d failed=%d\n",
		report.DiscoveredFiles,
		report.CreatedDocuments,
		report.UpdatedDocuments,
		report.SkippedFiles,
		report.FailedFiles,
	)

	for _, result := range report.Results {
		if result.Err != nil {
			fmt.Fprintf(os.Stdout, "[%s] %s error=%v\n", result.Action, result.Path, result.Err)
			continue
		}

		fmt.Fprintf(os.Stdout, "[%s] %s source_key=%s\n", result.Action, result.Path, result.SourceKey)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}
