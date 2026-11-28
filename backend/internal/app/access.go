package app

import (
	"context"
	"slices"

	"github.com/google/uuid"

	"github.com/thekogo/tutor-platform/backend/internal/store"
)

// Course Access: a Learner may study only the Courses the Owner granted them.
// The Owner implicitly has every Course. A Course without access is reported
// as not found, so its existence is not revealed.

// requireCourseAccess fails with not_found unless the Actor may study the Course.
func (s *Service) requireCourseAccess(ctx context.Context, a Actor, courseID uuid.UUID) error {
	if a.IsOwner {
		return nil
	}
	ok, err := s.q.HasCourseAccess(ctx, store.HasCourseAccessParams{LearnerID: a.LearnerID, CourseID: courseID})
	if err != nil {
		return err
	}
	if !ok {
		return notFound("course")
	}
	return nil
}

// accessibleGraph loads a Course's graph after checking the Actor's access.
func (s *Service) accessibleGraph(ctx context.Context, a Actor, courseID uuid.UUID) (*CourseGraph, error) {
	if err := s.requireCourseAccess(ctx, a, courseID); err != nil {
		return nil, err
	}
	return s.Graph(ctx, courseID)
}

// LearnerView is a Learner with their granted Courses (for the Owner).
type LearnerView struct {
	store.Learner
	CourseIDs []uuid.UUID // empty for the Owner, who has every Course
}

// ListLearners lists every Learner with their Course Access (Owner only).
func (s *Service) ListLearners(ctx context.Context, a Actor) ([]LearnerView, error) {
	if err := requireOwner(a); err != nil {
		return nil, err
	}
	ls, err := s.q.ListLearners(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListCourseAccess(ctx)
	if err != nil {
		return nil, err
	}
	byLearner := map[uuid.UUID][]uuid.UUID{}
	for _, r := range rows {
		byLearner[r.LearnerID] = append(byLearner[r.LearnerID], r.CourseID)
	}
	out := make([]LearnerView, 0, len(ls))
	for _, l := range ls {
		ids := byLearner[l.ID]
		if ids == nil {
			ids = []uuid.UUID{}
		}
		out = append(out, LearnerView{Learner: l, CourseIDs: ids})
	}
	return out, nil
}

// ValidateCourseIDs de-duplicates ids and checks every one is an existing
// Course. Also used by auth for the Courses an invite grants.
func ValidateCourseIDs(ctx context.Context, q *store.Queries, ids []uuid.UUID) ([]uuid.UUID, error) {
	ids = slices.Clone(ids)
	slices.SortFunc(ids, func(x, y uuid.UUID) int { return slices.Compare(x[:], y[:]) })
	ids = slices.Compact(ids)
	if len(ids) == 0 {
		return []uuid.UUID{}, nil
	}
	n, err := q.CountCoursesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if int(n) != len(ids) {
		return nil, E(KindInvalid, "unknown_course", "one or more Courses do not exist")
	}
	return ids, nil
}

// SetLearnerCourses replaces the set of Courses a Learner may study (Owner only).
func (s *Service) SetLearnerCourses(ctx context.Context, a Actor, learnerID uuid.UUID, courseIDs []uuid.UUID) (LearnerView, error) {
	if err := requireOwner(a); err != nil {
		return LearnerView{}, err
	}
	var out LearnerView
	err := s.tx(ctx, func(q *store.Queries) error {
		l, err := q.GetLearner(ctx, learnerID)
		if err != nil {
			return mapNotFound(err, "learner")
		}
		if l.IsOwner {
			return E(KindInvalid, "owner_has_all_courses", "the Owner always has access to every Course")
		}
		ids, err := ValidateCourseIDs(ctx, q, courseIDs)
		if err != nil {
			return err
		}
		if err := q.DeleteLearnerCourseAccess(ctx, learnerID); err != nil {
			return err
		}
		if len(ids) > 0 {
			if err := q.GrantCourseAccess(ctx, store.GrantCourseAccessParams{LearnerID: learnerID, CourseIds: ids}); err != nil {
				return err
			}
		}
		out = LearnerView{Learner: l, CourseIDs: ids}
		return nil
	})
	return out, err
}
