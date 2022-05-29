package crawler

import (
	"strconv"
	"time"

	"github.com/ymakhloufi/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

var _ SiteCrawler = &DummyCrawler{}

type DummyCrawler struct {
	logger *zap.Logger
}

func NewDummyCrawler(logger *zap.Logger) *DummyCrawler {
	return &DummyCrawler{logger: logger}
}

func (d DummyCrawler) Crawl(sets chan<- model.InterestSet) {
	for i := 1; i < 5; i++ {
		d.logger.Info("dummy crawler crawling " + strconv.Itoa(i))
		sets <- model.InterestSet{Term: model.Term("foo " + strconv.Itoa(i))}
		time.Sleep(time.Second * 3)
	}
}
