package main

import (
	"github.com/ymakhloufi/bolan-compare/internal/app/crawler"
	"github.com/ymakhloufi/bolan-compare/internal/pkg/store"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	noErr(err)

	crawlers := []crawler.SiteCrawler{
		//crawler.NewDummyCrawler(logger.Named("DummyCrawler")),
		crawler.NewDanskeBankCrawler(logger.Named("DanskeBankCrawler")),
	}

	pgStore := store.NewPostgres(nil, logger.Named("PG Store"))
	svc := crawler.NewService(pgStore, crawlers, logger.Named("Crawler Svc"))

	svc.Crawl()
}

func noErr(err error) {
	if err != nil {
		panic("failed to initialize something important: " + err.Error())
	}
}
