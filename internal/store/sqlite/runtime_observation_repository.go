package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/netlab/netlab/internal/domain"
)

func (r *Repositories) SaveCaptureObservation(ctx context.Context, capture domain.Capture) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error { return saveCaptureObservation(ctx, tx, capture) })
}

func saveCaptureObservation(ctx context.Context, tx *sql.Tx, capture domain.Capture) error {
	lastError, _ := json.Marshal(capture.LastError)
	_, err := tx.ExecContext(ctx, `INSERT INTO captures(id,laboratory_id,source_type,source_id,purpose,parent_resource_id,filter,format,state,retain,max_bytes,bytes_written,packets,truncated,artifact_id,artifact_url,expires_at,completion_reason,created_at,started_at,finished_at,last_error_json)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET laboratory_id=excluded.laboratory_id,source_type=excluded.source_type,source_id=excluded.source_id,purpose=excluded.purpose,parent_resource_id=excluded.parent_resource_id,filter=excluded.filter,format=excluded.format,state=excluded.state,retain=excluded.retain,max_bytes=excluded.max_bytes,bytes_written=excluded.bytes_written,packets=excluded.packets,truncated=excluded.truncated,artifact_id=excluded.artifact_id,artifact_url=excluded.artifact_url,expires_at=excluded.expires_at,completion_reason=excluded.completion_reason,started_at=excluded.started_at,finished_at=excluded.finished_at,last_error_json=excluded.last_error_json`,
		capture.ID, nullable(string(capture.LaboratoryID)), capture.SourceType, capture.SourceID, capture.Purpose, nullable(string(capture.ParentResourceID)), capture.Filter, capture.Format, capture.State, capture.Retain, capture.MaxBytes, capture.BytesWritten, capture.Packets, capture.Truncated, nullable(string(capture.ArtifactID)), capture.ArtifactURL, formatTime(capture.ExpiresAt), capture.CompletionReason, capture.CreatedAt.Format(time.RFC3339Nano), formatTime(capture.StartedAt), formatTime(capture.FinishedAt), nullableBytes(lastError, capture.LastError != nil))
	return err
}

func (r *Repositories) GetCaptureObservation(ctx context.Context, id domain.ID) (domain.Capture, error) {
	var capture domain.Capture
	var laboratoryID, parentID, artifactID, expiresAt, startedAt, finishedAt sql.NullString
	var createdAt string
	var lastError []byte
	err := r.database.DB.QueryRowContext(ctx, `SELECT id,laboratory_id,source_type,source_id,purpose,parent_resource_id,filter,format,state,retain,max_bytes,bytes_written,packets,truncated,artifact_id,artifact_url,expires_at,completion_reason,created_at,started_at,finished_at,COALESCE(last_error_json,'') FROM captures WHERE id=?`, id).Scan(
		&capture.ID, &laboratoryID, &capture.SourceType, &capture.SourceID, &capture.Purpose, &parentID, &capture.Filter, &capture.Format, &capture.State, &capture.Retain, &capture.MaxBytes, &capture.BytesWritten, &capture.Packets, &capture.Truncated, &artifactID, &capture.ArtifactURL, &expiresAt, &capture.CompletionReason, &createdAt, &startedAt, &finishedAt, &lastError)
	if err == sql.ErrNoRows {
		return capture, ErrNotFound
	}
	if err != nil {
		return capture, err
	}
	capture.LaboratoryID, capture.ParentResourceID, capture.ArtifactID = domain.ID(laboratoryID.String), domain.ID(parentID.String), domain.ID(artifactID.String)
	capture.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	capture.ExpiresAt, capture.StartedAt, capture.FinishedAt = parseNullTime(expiresAt), parseNullTime(startedAt), parseNullTime(finishedAt)
	if len(lastError) > 0 {
		capture.LastError = &domain.Problem{}
		_ = json.Unmarshal(lastError, capture.LastError)
	}
	return capture, nil
}

func (r *Repositories) ListCaptureObservations(ctx context.Context, laboratoryID domain.ID) ([]domain.Capture, error) {
	rows, err := r.database.DB.QueryContext(ctx, `SELECT id FROM captures WHERE laboratory_id=? ORDER BY created_at,id`, laboratoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []domain.ID
	for rows.Next() {
		var id domain.ID
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	captures := make([]domain.Capture, 0, len(ids))
	for _, id := range ids {
		capture, getErr := r.GetCaptureObservation(ctx, id)
		if getErr != nil {
			return nil, getErr
		}
		captures = append(captures, capture)
	}
	return captures, nil
}

func (r *Repositories) SaveTrafficFilterObservation(ctx context.Context, filter domain.TrafficFilter) error {
	return r.database.Write(ctx, func(tx *sql.Tx) error { return saveTrafficFilterObservation(ctx, tx, filter) })
}

func saveTrafficFilterObservation(ctx context.Context, tx *sql.Tx, filter domain.TrafficFilter) error {
	interfaceIDs, _ := json.Marshal(filter.InterfaceIDs)
	linkIDs, _ := json.Marshal(filter.LinkIDs)
	objectLinkIDs, _ := json.Marshal(filter.NetworkObjectLinkIDs)
	observations, _ := json.Marshal(filter.Observations)
	lastError, _ := json.Marshal(filter.LastError)
	if _, err := tx.ExecContext(ctx, `INSERT INTO traffic_filters(id,laboratory_id,expression,color,state,max_observations,interface_ids_json,link_ids_json,network_object_link_ids_json,observations_json,created_at,finished_at,last_error_json)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET expression=excluded.expression,color=excluded.color,state=excluded.state,max_observations=excluded.max_observations,interface_ids_json=excluded.interface_ids_json,link_ids_json=excluded.link_ids_json,network_object_link_ids_json=excluded.network_object_link_ids_json,observations_json=excluded.observations_json,finished_at=excluded.finished_at,last_error_json=excluded.last_error_json`,
		filter.ID, filter.LaboratoryID, filter.Expression, filter.Color, filter.State, filter.MaxObservations, interfaceIDs, linkIDs, objectLinkIDs, observations, filter.CreatedAt.Format(time.RFC3339Nano), formatTime(filter.FinishedAt), nullableBytes(lastError, filter.LastError != nil)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM traffic_observations WHERE traffic_filter_id=?`, filter.ID); err != nil {
		return err
	}
	for _, observation := range filter.Observations {
		if _, err := tx.ExecContext(ctx, `INSERT INTO traffic_observations(traffic_filter_id,fingerprint,resource_type,resource_id,interface_id,link_id,network_object_link_id,direction,source_address,destination_address,source_mac,destination_mac,packet_role,first_seen_at,last_seen_at,packet_count,byte_count) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			filter.ID, observation.Fingerprint, observation.ResourceType, observation.ResourceID, nullable(string(observation.InterfaceID)), nullable(string(observation.LinkID)), nullable(string(observation.NetworkObjectLinkID)), observation.Direction, observation.SourceAddress, observation.DestinationAddress, observation.SourceMAC, observation.DestinationMAC, observation.PacketRole, observation.FirstSeen.Format(time.RFC3339Nano), observation.LastSeen.Format(time.RFC3339Nano), observation.Count, observation.Bytes); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repositories) GetTrafficFilterObservation(ctx context.Context, id domain.ID) (domain.TrafficFilter, error) {
	var filter domain.TrafficFilter
	var interfaceIDs, linkIDs, objectLinkIDs, lastError []byte
	var createdAt string
	var finishedAt sql.NullString
	err := r.database.DB.QueryRowContext(ctx, `SELECT id,laboratory_id,expression,color,state,max_observations,interface_ids_json,link_ids_json,network_object_link_ids_json,created_at,finished_at,COALESCE(last_error_json,'') FROM traffic_filters WHERE id=?`, id).Scan(&filter.ID, &filter.LaboratoryID, &filter.Expression, &filter.Color, &filter.State, &filter.MaxObservations, &interfaceIDs, &linkIDs, &objectLinkIDs, &createdAt, &finishedAt, &lastError)
	if err == sql.ErrNoRows {
		return filter, ErrNotFound
	}
	if err != nil {
		return filter, err
	}
	_ = json.Unmarshal(interfaceIDs, &filter.InterfaceIDs)
	_ = json.Unmarshal(linkIDs, &filter.LinkIDs)
	_ = json.Unmarshal(objectLinkIDs, &filter.NetworkObjectLinkIDs)
	filter.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	filter.FinishedAt = parseNullTime(finishedAt)
	if len(lastError) > 0 {
		filter.LastError = &domain.Problem{}
		_ = json.Unmarshal(lastError, filter.LastError)
	}
	rows, err := r.database.DB.QueryContext(ctx, `SELECT fingerprint,resource_type,resource_id,COALESCE(interface_id,''),COALESCE(link_id,''),COALESCE(network_object_link_id,''),direction,source_address,destination_address,source_mac,destination_mac,packet_role,first_seen_at,last_seen_at,packet_count,byte_count FROM traffic_observations WHERE traffic_filter_id=? ORDER BY first_seen_at,fingerprint`, id)
	if err != nil {
		return filter, err
	}
	defer rows.Close()
	for rows.Next() {
		var observation domain.TrafficObservation
		var firstSeen, lastSeen string
		if err = rows.Scan(&observation.Fingerprint, &observation.ResourceType, &observation.ResourceID, &observation.InterfaceID, &observation.LinkID, &observation.NetworkObjectLinkID, &observation.Direction, &observation.SourceAddress, &observation.DestinationAddress, &observation.SourceMAC, &observation.DestinationMAC, &observation.PacketRole, &firstSeen, &lastSeen, &observation.Count, &observation.Bytes); err != nil {
			return filter, err
		}
		observation.FirstSeen, _ = time.Parse(time.RFC3339Nano, firstSeen)
		observation.LastSeen, _ = time.Parse(time.RFC3339Nano, lastSeen)
		filter.Observations = append(filter.Observations, observation)
	}
	return filter, rows.Err()
}
