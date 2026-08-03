package query

import "github.com/netlab/netlab/internal/domain"

type ReleaseService struct {
	identity domain.ReleaseIdentity
}

func NewReleaseService(identity domain.ReleaseIdentity) *ReleaseService {
	return &ReleaseService{identity: identity}
}

func (s *ReleaseService) Get() domain.ReleaseIdentity {
	if s == nil {
		return domain.ReleaseIdentity{}
	}
	return s.identity
}
