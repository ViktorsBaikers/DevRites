//go:build unix

package state

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

type fifoObservationResult struct {
	observation    *WorkspaceObservation
	observationErr error
	setupErr       error
	fifoCreated    bool
}

func TestWorkspaceObservationFIFORegularSubstitutionDoesNotWaitForWriter(t *testing.T) {
	root, slug := newObservationWorkspace(t)
	writeObservationArtifact(t, root, slug, LedgerFile, []byte("- Phase: build\n"))
	writeObservationArtifact(t, root, slug, "brief.md", []byte("regular\n"))
	artifactPath := filepath.Join(root, "work", slug, "brief.md")

	result := make(chan fifoObservationResult, 1)
	go func() {
		var setupErr error
		fifoCreated := false
		observation, observationErr := observeWorkspace(root, slug, func(stage observationStage, artifact ArtifactPath) error {
			if stage != observationBeforeOpen || artifact != "brief.md" {
				return nil
			}
			if err := os.Remove(artifactPath); err != nil {
				setupErr = err
				return nil
			}
			if err := syscall.Mkfifo(artifactPath, 0o600); err != nil {
				setupErr = err
				return nil
			}
			fifoCreated = true
			return nil
		})
		result <- fifoObservationResult{
			observation:    observation,
			observationErr: observationErr,
			setupErr:       setupErr,
			fifoCreated:    fifoCreated,
		}
	}()

	select {
	case got := <-result:
		if got.setupErr != nil {
			t.Fatalf("FIFO setup failed: %v", got.setupErr)
		}
		if !got.fifoCreated {
			t.Fatal("FIFO setup reported success without creating the FIFO")
		}
		if got.observation != nil {
			t.Fatalf("observation = %+v, want nil", got.observation)
		}
		assertObservationFailure(t, got.observationErr, ObservationConcurrentChange, "workspace observation: concurrent_change: workspace changed during acquisition; retry")
	case <-time.After(2 * time.Second):
		t.Fatal("ObserveWorkspace waited for a FIFO writer")
	}
}
