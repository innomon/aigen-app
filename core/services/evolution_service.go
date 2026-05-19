package services

import (
	"context"
	"fmt"
	"time"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/innomon/aigen-app/utils/datamodels"
)

type EvolutionService struct {
	dao           relationdbdao.IPrimaryDao
	schemaService ISchemaService
	manifests     map[string]descriptors.EvolutionManifest // Key is BizDef name
}

func NewEvolutionService(dao relationdbdao.IPrimaryDao, schemaService ISchemaService) *EvolutionService {
	return &EvolutionService{
		dao:           dao,
		schemaService: schemaService,
		manifests:     make(map[string]descriptors.EvolutionManifest),
	}
}

func (s *EvolutionService) RegisterManifest(bizdefName string, manifest descriptors.EvolutionManifest) {
	s.manifests[bizdefName] = manifest
}

// EvolveRecord applies all necessary transformations to bring a record to the latest version
func (s *EvolutionService) EvolveRecord(entityName string, rec map[string]interface{}, meta *datamodels.MetaData) (map[string]interface{}, bool, error) {
	// 1. Find the manifest for this entity
	var entityTimeline map[string]descriptors.EntityVersion
	found := false
	for _, m := range s.manifests {
		if timeline, ok := m[entityName]; ok {
			entityTimeline = timeline
			found = true
			break
		}
	}

	if !found || len(entityTimeline) == 0 {
		return rec, false, nil
	}

	// 2. Determine the path of versions to apply
	// We need to sort versions by date
	type versionInfo struct {
		ver string
		ev  descriptors.EntityVersion
	}
	var timeline []versionInfo
	for ver, ev := range entityTimeline {
		timeline = append(timeline, versionInfo{ver, ev})
	}

	// Sort timeline by date
	for i := 0; i < len(timeline); i++ {
		for j := i + 1; j < len(timeline); j++ {
			if timeline[i].ev.Date.After(timeline[j].ev.Date) {
				timeline[i], timeline[j] = timeline[j], timeline[i]
			}
		}
	}

	// 3. Apply transformations sequentially
	modified := false
	recordVersionDate := time.Time{}
	if meta.SchemaVersionDate != "" {
		t, err := time.Parse(time.RFC3339, meta.SchemaVersionDate)
		if err == nil {
			recordVersionDate = t
		}
	}

	for _, v := range timeline {
		if v.ev.Date.After(recordVersionDate) {
			s.applyActions(rec, v.ev.Actions)
			meta.SchemaVersion = v.ver
			meta.SchemaVersionDate = v.ev.Date.Format(time.RFC3339)
			modified = true
		}
	}

	return rec, modified, nil
}

func (s *EvolutionService) applyActions(rec map[string]interface{}, actions []descriptors.EvolutionAction) {
	for _, action := range actions {
		switch action.Action {
		case "rename":
			if val, ok := rec[action.From]; ok {
				rec[action.To] = val
				delete(rec, action.From)
			}
		case "add":
			if _, ok := rec[action.Field]; !ok {
				rec[action.Field] = action.Default
			}
		case "drop":
			delete(rec, action.Field)
		case "transform":
			// Placeholder for more complex logic
			// In a real system, this might involve calling a registered Go function
		}
	}
}

// ScrubEntity performs a batch migration of all records for an entity
func (s *EvolutionService) ScrubEntity(ctx context.Context, entityName string, batchSize int) (int, int, error) {
	namespace := fmt.Sprintf("aigen.bizdef.entities.%s", entityName)
	
	offset := 0
	totalUpgraded := 0
	totalFailed := 0

	for {
		limit := fmt.Sprintf("%d", batchSize)
		offStr := fmt.Sprintf("%d", offset)
		recs, _, err := s.dao.List(ctx, namespace, nil, datamodels.Pagination{Limit: &limit, Offset: &offStr}, nil)
		if err != nil {
			return totalUpgraded, totalFailed, fmt.Errorf("failed to list records for %s: %w", entityName, err)
		}

		if len(recs) == 0 {
			break
		}

		for _, r := range recs {
			recData := r.Rec.(map[string]interface{})
			// Use a copy of metadata to avoid modifying the one in the record before we decide to save
			metaCopy := r.MetaData
			
			_, modified, err := s.EvolveRecord(entityName, recData, &metaCopy)
			if err != nil {
				totalFailed++
				continue
			}

			if modified {
				r.Rec = recData
				r.MetaData = metaCopy
				// Revision will be incremented inside SaveConditional
				if err := s.dao.SaveConditional(ctx, r, r.MetaData.Revision); err != nil {
					// Likely a concurrency conflict, skip and it will be picked up in next run or JIT
					totalFailed++
				} else {
					totalUpgraded++
				}
			}
		}

		if len(recs) < batchSize {
			break
		}
		offset += batchSize
	}

	return totalUpgraded, totalFailed, nil
}
