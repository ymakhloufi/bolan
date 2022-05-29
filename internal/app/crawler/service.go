package crawler

import (
	"sync"

	"github.com/ymakhloufi/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

type Store interface {
	UpsertInterestSet(set model.InterestSet) error
}

type SiteCrawler interface {
	Crawl(chan<- model.InterestSet)
}

type Service struct {
	store    Store
	crawlers []SiteCrawler
	logger   *zap.Logger
}

func NewService(store Store, crawlers []SiteCrawler, logger *zap.Logger) *Service {
	return &Service{
		store:    store,
		crawlers: crawlers,
		logger:   logger,
	}
}

func (s Service) Crawl() {
	var wg sync.WaitGroup
	objChan := make(chan model.InterestSet)

	for _, c := range s.crawlers {
		wg.Add(1)
		go func(c SiteCrawler) {
			defer wg.Done()
			c.Crawl(objChan)
		}(c)
	}

	go s.recv(objChan)

	wg.Wait()
	s.logger.Info("all crawlers finished, closing channels")
	close(objChan)
}

func (s Service) recv(c <-chan model.InterestSet) {
	s.logger.Info("starting crawler receiver")

	for set := range c {
		if err := s.store.UpsertInterestSet(set); err != nil {
			s.logger.Error("failed to upsert interestSet", zap.Any("interestSet", set), zap.Error(err))
		}

		s.logger.Info("successfully upserted interestSet", zap.Any("interestSet", set))
	}
}
