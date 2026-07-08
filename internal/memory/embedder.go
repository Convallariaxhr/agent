// internal/memory/embedder.go
package memory

// Embedder is a placeholder for the embedding model integration.
// In production, this would use a local model (e.g., all-MiniLM-L6-v2 via ONNX)
// or a remote embedding API.
type Embedder struct{}

// Embed converts text to a vector. Placeholder: returns nil.
func (e *Embedder) Embed(text string) ([]float32, error) {
	// Placeholder: in production, run inference with a local model
	return nil, nil
}