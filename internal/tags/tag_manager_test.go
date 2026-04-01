package tags

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper to create a TagManager with in-memory storage
func newTestTagManager() *TagManager {
	return NewTagManager(NewMemoryTagStorage())
}

// helper to create a tag and return it
func createTestTag(t *testing.T, m *TagManager, name, color string) *Tag {
	t.Helper()
	tag, err := m.CreateTag(&TagCreateRequest{Name: name, Color: color})
	require.NoError(t, err)
	require.NotNil(t, tag)
	return tag
}

func TestTagManager_CreateTag(t *testing.T) {
	tests := []struct {
		name      string
		request   *TagCreateRequest
		expectErr bool
		errVal    error
	}{
		{
			name:      "successful creation",
			request:   &TagCreateRequest{Name: "production", Color: "#FF0000"},
			expectErr: false,
		},
		{
			name:      "successful creation without color",
			request:   &TagCreateRequest{Name: "staging"},
			expectErr: false,
		},
		{
			name:      "empty name",
			request:   &TagCreateRequest{Name: ""},
			expectErr: true,
			errVal:    ErrTagNameRequired,
		},
		{
			name:      "name too long",
			request:   &TagCreateRequest{Name: string(make([]byte, 101))},
			expectErr: true,
			errVal:    ErrTagNameTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestTagManager()
			tag, err := m.CreateTag(tt.request)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errVal != nil {
					assert.ErrorIs(t, err, tt.errVal)
				}
				assert.Nil(t, tag)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, tag)
			assert.NotEmpty(t, tag.ID)
			assert.Equal(t, tt.request.Name, tag.Name)
			assert.Equal(t, tt.request.Color, tag.Color)
			assert.False(t, tag.CreatedAt.IsZero())
			assert.False(t, tag.UpdatedAt.IsZero())
		})
	}
}

func TestTagManager_CreateTag_DuplicateName(t *testing.T) {
	m := newTestTagManager()
	createTestTag(t, m, "production", "#FF0000")

	_, err := m.CreateTag(&TagCreateRequest{Name: "production", Color: "#00FF00"})
	assert.ErrorIs(t, err, ErrTagNameExists)
}

func TestTagManager_GetTag(t *testing.T) {
	m := newTestTagManager()
	created := createTestTag(t, m, "test-tag", "#AABBCC")

	t.Run("existing tag", func(t *testing.T) {
		tag, err := m.GetTag(created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, tag.ID)
		assert.Equal(t, "test-tag", tag.Name)
	})

	t.Run("non-existent tag", func(t *testing.T) {
		_, err := m.GetTag("nonexistent")
		assert.ErrorIs(t, err, ErrTagNotFound)
	})
}

func TestTagManager_GetTagByName(t *testing.T) {
	m := newTestTagManager()
	created := createTestTag(t, m, "unique-tag", "#112233")

	t.Run("existing name", func(t *testing.T) {
		tag, err := m.GetTagByName("unique-tag")
		require.NoError(t, err)
		assert.Equal(t, created.ID, tag.ID)
	})

	t.Run("non-existent name", func(t *testing.T) {
		_, err := m.GetTagByName("does-not-exist")
		assert.ErrorIs(t, err, ErrTagNotFound)
	})
}

func TestTagManager_ListTags(t *testing.T) {
	m := newTestTagManager()
	createTestTag(t, m, "alpha", "#111111")
	createTestTag(t, m, "beta", "#222222")
	createTestTag(t, m, "gamma", "#333333")

	t.Run("list all", func(t *testing.T) {
		tags, total, err := m.ListTags(TagListFilters{})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, tags, 3)
	})

	t.Run("with search filter", func(t *testing.T) {
		tags, total, err := m.ListTags(TagListFilters{Search: "alp"})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, tags, 1)
		assert.Equal(t, "alpha", tags[0].Name)
	})

	t.Run("with pagination", func(t *testing.T) {
		tags, total, err := m.ListTags(TagListFilters{Limit: 2, Offset: 0})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, tags, 2)
	})

	t.Run("offset beyond total", func(t *testing.T) {
		tags, total, err := m.ListTags(TagListFilters{Offset: 10})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, tags, 0)
	})
}

func TestTagManager_UpdateTag(t *testing.T) {
	m := newTestTagManager()
	original := createTestTag(t, m, "old-name", "#000000")

	t.Run("update name", func(t *testing.T) {
		updated, err := m.UpdateTag(original.ID, &TagUpdateRequest{Name: "new-name"})
		require.NoError(t, err)
		assert.Equal(t, "new-name", updated.Name)
		assert.Equal(t, "#000000", updated.Color)
	})

	t.Run("update color", func(t *testing.T) {
		updated, err := m.UpdateTag(original.ID, &TagUpdateRequest{Color: "#FFFFFF"})
		require.NoError(t, err)
		assert.Equal(t, "#FFFFFF", updated.Color)
	})

	t.Run("update non-existent tag", func(t *testing.T) {
		_, err := m.UpdateTag("nonexistent", &TagUpdateRequest{Name: "x"})
		assert.ErrorIs(t, err, ErrTagNotFound)
	})

	t.Run("update with too long name", func(t *testing.T) {
		_, err := m.UpdateTag(original.ID, &TagUpdateRequest{Name: string(make([]byte, 101))})
		assert.ErrorIs(t, err, ErrTagNameTooLong)
	})
}

func TestTagManager_DeleteTag(t *testing.T) {
	m := newTestTagManager()
	tag := createTestTag(t, m, "to-delete", "#FF0000")

	t.Run("delete existing tag", func(t *testing.T) {
		err := m.DeleteTag(tag.ID)
		require.NoError(t, err)

		_, err = m.GetTag(tag.ID)
		assert.ErrorIs(t, err, ErrTagNotFound)
	})

	t.Run("delete non-existent tag", func(t *testing.T) {
		err := m.DeleteTag("nonexistent")
		assert.ErrorIs(t, err, ErrTagNotFound)
	})
}

func TestTagManager_DeleteTag_InUse(t *testing.T) {
	m := newTestTagManager()
	tag := createTestTag(t, m, "in-use-tag", "#FF0000")

	err := m.AddWorkflowTag("workflow-1", tag.ID)
	require.NoError(t, err)

	err = m.DeleteTag(tag.ID)
	assert.ErrorIs(t, err, ErrTagInUse)
}

func TestTagManager_WorkflowTags(t *testing.T) {
	m := newTestTagManager()
	tag1 := createTestTag(t, m, "tag-1", "#111111")
	tag2 := createTestTag(t, m, "tag-2", "#222222")

	t.Run("add workflow tag", func(t *testing.T) {
		err := m.AddWorkflowTag("wf-1", tag1.ID)
		require.NoError(t, err)
	})

	t.Run("add non-existent tag to workflow", func(t *testing.T) {
		err := m.AddWorkflowTag("wf-1", "nonexistent")
		assert.ErrorIs(t, err, ErrTagNotFound)
	})

	t.Run("get workflow tags", func(t *testing.T) {
		err := m.AddWorkflowTag("wf-1", tag2.ID)
		require.NoError(t, err)

		tags, err := m.GetWorkflowTags("wf-1")
		require.NoError(t, err)
		assert.Len(t, tags, 2)
	})

	t.Run("get workflow tags for untagged workflow", func(t *testing.T) {
		tags, err := m.GetWorkflowTags("wf-no-tags")
		require.NoError(t, err)
		assert.Len(t, tags, 0)
	})

	t.Run("remove workflow tag", func(t *testing.T) {
		err := m.RemoveWorkflowTag("wf-1", tag1.ID)
		require.NoError(t, err)

		tags, err := m.GetWorkflowTags("wf-1")
		require.NoError(t, err)
		assert.Len(t, tags, 1)
		assert.Equal(t, tag2.ID, tags[0].ID)
	})

	t.Run("get tag workflows", func(t *testing.T) {
		_ = m.AddWorkflowTag("wf-2", tag2.ID)
		workflows, err := m.GetTagWorkflows(tag2.ID)
		require.NoError(t, err)
		assert.Len(t, workflows, 2)
	})
}

func TestTagManager_SetWorkflowTags(t *testing.T) {
	m := newTestTagManager()
	tag1 := createTestTag(t, m, "set-tag-1", "#111111")
	tag2 := createTestTag(t, m, "set-tag-2", "#222222")
	tag3 := createTestTag(t, m, "set-tag-3", "#333333")

	// Add initial tags
	_ = m.AddWorkflowTag("wf-set", tag1.ID)
	_ = m.AddWorkflowTag("wf-set", tag2.ID)

	t.Run("set workflow tags replaces all", func(t *testing.T) {
		err := m.SetWorkflowTags("wf-set", []string{tag3.ID})
		require.NoError(t, err)

		tags, err := m.GetWorkflowTags("wf-set")
		require.NoError(t, err)
		assert.Len(t, tags, 1)
		assert.Equal(t, tag3.ID, tags[0].ID)
	})

	t.Run("set with non-existent tag", func(t *testing.T) {
		err := m.SetWorkflowTags("wf-set", []string{"nonexistent"})
		assert.Error(t, err)
	})

	t.Run("set empty tags clears all", func(t *testing.T) {
		err := m.SetWorkflowTags("wf-set", []string{})
		require.NoError(t, err)

		tags, err := m.GetWorkflowTags("wf-set")
		require.NoError(t, err)
		assert.Len(t, tags, 0)
	})
}

func TestTagCreateRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		request   TagCreateRequest
		expectErr error
	}{
		{
			name:      "valid request",
			request:   TagCreateRequest{Name: "valid"},
			expectErr: nil,
		},
		{
			name:      "empty name",
			request:   TagCreateRequest{Name: ""},
			expectErr: ErrTagNameRequired,
		},
		{
			name:      "name too long",
			request:   TagCreateRequest{Name: string(make([]byte, 101))},
			expectErr: ErrTagNameTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.expectErr != nil {
				assert.ErrorIs(t, err, tt.expectErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTagUpdateRequest_Validate(t *testing.T) {
	tests := []struct {
		name      string
		request   TagUpdateRequest
		expectErr error
	}{
		{
			name:      "valid with name",
			request:   TagUpdateRequest{Name: "valid"},
			expectErr: nil,
		},
		{
			name:      "valid empty (no changes)",
			request:   TagUpdateRequest{},
			expectErr: nil,
		},
		{
			name:      "name too long",
			request:   TagUpdateRequest{Name: string(make([]byte, 101))},
			expectErr: ErrTagNameTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.expectErr != nil {
				assert.ErrorIs(t, err, tt.expectErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
