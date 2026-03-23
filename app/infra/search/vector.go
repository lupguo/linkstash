package search

import (
	"log/slog"
	"sort"
	"sync"

	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/domain/repos"
	"github.com/lupguo/linkstash/app/infra/llm"
)

// VectorResult holds a single vector search result.
type VectorResult struct {
	ID         uint
	Similarity float64
}

// VectorSearch provides in-memory cosine similarity search over embeddings.
type VectorSearch struct {
	mu            sync.RWMutex
	cache         map[uint][]float32 // urlID -> vector
	embeddingRepo repos.EmbeddingRepo
}

func NewVectorSearch(embeddingRepo repos.EmbeddingRepo) *VectorSearch {
	return &VectorSearch{
		cache:         make(map[uint][]float32),
		embeddingRepo: embeddingRepo,
	}
}

// LoadAll loads all embeddings from DB into memory cache. Call at startup.
func (vs *VectorSearch) LoadAll() error {
	embeddings, err := vs.embeddingRepo.GetAll()
	if err != nil {
		return err
	}

	vs.mu.Lock()
	defer vs.mu.Unlock()
	for _, e := range embeddings {
		vs.cache[e.URLID] = llm.BytesToFloat32s(e.Vector)
	}
	slog.Info("loaded embeddings into memory cache", "component", "vector_search", "count", len(vs.cache))
	return nil
}

// UpdateCache adds or updates a single embedding in the cache.
func (vs *VectorSearch) UpdateCache(e *entity.Embedding) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	vs.cache[e.URLID] = llm.BytesToFloat32s(e.Vector)
}

// RemoveFromCache removes an embedding from the cache.
func (vs *VectorSearch) RemoveFromCache(urlID uint) {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	delete(vs.cache, urlID)
}

// Search finds the top-N most similar vectors to the query vector.
func (vs *VectorSearch) Search(queryVector []float32, limit int) []VectorResult {
	if limit <= 0 {
		limit = 50
	}

	vs.mu.RLock()
	defer vs.mu.RUnlock()

	results := make([]VectorResult, 0, len(vs.cache))
	for id, vec := range vs.cache {
		sim := llm.CosineSimilarity(queryVector, vec)
		results = append(results, VectorResult{ID: id, Similarity: sim})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if len(results) > limit {
		results = results[:limit]
	}
	return results
}
