package services

import (
	"context"
	"errors"
	"testing"

	"radio-backend/internal/domain"
	"radio-backend/internal/i18n"
	"radio-backend/internal/infrastructure/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	// Inicializar logger para tests
	logger.Init("text", "error")
}

// MockTranslationRepository es un mock del TranslationRepository
type MockTranslationRepository struct {
	mock.Mock
}

func (m *MockTranslationRepository) Create(translation *domain.StationTranslation) error {
	args := m.Called(translation)
	return args.Error(0)
}

func (m *MockTranslationRepository) Get(stationID string, languageCode i18n.Language) (*domain.StationTranslation, error) {
	args := m.Called(stationID, languageCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.StationTranslation), args.Error(1)
}

func (m *MockTranslationRepository) Update(translation *domain.StationTranslation) error {
	args := m.Called(translation)
	return args.Error(0)
}

func (m *MockTranslationRepository) Delete(stationID string, languageCode i18n.Language) error {
	args := m.Called(stationID, languageCode)
	return args.Error(0)
}

func (m *MockTranslationRepository) ListByStation(stationID string) ([]*domain.StationTranslation, error) {
	args := m.Called(stationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.StationTranslation), args.Error(1)
}

func (m *MockTranslationRepository) ListByLanguage(languageCode i18n.Language, limit, offset int) ([]*domain.StationTranslation, error) {
	args := m.Called(languageCode, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.StationTranslation), args.Error(1)
}

func (m *MockTranslationRepository) Exists(stationID string, languageCode i18n.Language) (bool, error) {
	args := m.Called(stationID, languageCode)
	return args.Bool(0), args.Error(1)
}

func (m *MockTranslationRepository) BulkCreate(translations []*domain.StationTranslation) error {
	args := m.Called(translations)
	return args.Error(0)
}

func (m *MockTranslationRepository) GetAvailableLanguages(stationID string) ([]i18n.Language, error) {
	args := m.Called(stationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]i18n.Language), args.Error(1)
}

func (m *MockTranslationRepository) GetByStationIDs(stationIDs []string, languageCode i18n.Language) (map[string]*domain.StationTranslation, error) {
	args := m.Called(stationIDs, languageCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*domain.StationTranslation), args.Error(1)
}

// MockStationRepository es un mock del StationRepository
type MockStationRepository struct {
	mock.Mock
}

func (m *MockStationRepository) FindByID(ctx context.Context, id string) (*domain.Station, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Station), args.Error(1)
}

func (m *MockStationRepository) FindPopular(ctx context.Context, limit int, country string) ([]domain.Station, error) {
	args := m.Called(limit, country)
	return args.Get(0).([]domain.Station), args.Error(1)
}

func (m *MockStationRepository) Search(ctx context.Context, query string, limit int) ([]domain.Station, error) {
	args := m.Called(query, limit)
	return args.Get(0).([]domain.Station), args.Error(1)
}

func TestTranslationService_CreateTranslation(t *testing.T) {
	t.Run("success - create valid translation", func(t *testing.T) {
		// Arrange
		mockTranslationRepo := new(MockTranslationRepository)
		mockStationRepo := new(MockStationRepository)
		service := NewTranslationService(mockTranslationRepo, mockStationRepo)

		station := &domain.Station{ID: "station-1", Name: "Test Radio"}
		req := &domain.CreateTranslationRequest{
			StationID:    "station-1",
			LanguageCode: "en",
			Title:        "Test Radio Live",
			Description:  "Listen to Test Radio live",
			Keywords:     []string{"radio", "live"},
		}

		mockStationRepo.On("FindByID", "station-1").Return(station, nil)
		mockTranslationRepo.On("Create", mock.AnythingOfType("*domain.StationTranslation")).Return(nil)

		// Act
		result, err := service.CreateTranslation(req)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "station-1", result.StationID)
		assert.Equal(t, i18n.LanguageEN, result.LanguageCode)
		mockStationRepo.AssertExpectations(t)
		mockTranslationRepo.AssertExpectations(t)
	})

	t.Run("error - station not found", func(t *testing.T) {
		// Arrange
		mockTranslationRepo := new(MockTranslationRepository)
		mockStationRepo := new(MockStationRepository)
		service := NewTranslationService(mockTranslationRepo, mockStationRepo)

		req := &domain.CreateTranslationRequest{
			StationID:    "nonexistent",
			LanguageCode: "en",
			Title:        "Test",
			Description:  "Test description",
		}

		mockStationRepo.On("FindByID", "nonexistent").Return(nil, domain.ErrStationNotFound)

		// Act
		result, err := service.CreateTranslation(context.Background(), req)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrStationNotFound, err)
		mockStationRepo.AssertExpectations(t)
	})

	t.Run("error - invalid title", func(t *testing.T) {
		// Arrange
		mockTranslationRepo := new(MockTranslationRepository)
		mockStationRepo := new(MockStationRepository)
		service := NewTranslationService(mockTranslationRepo, mockStationRepo)

		station := &domain.Station{ID: "station-1", Name: "Test Radio"}
		req := &domain.CreateTranslationRequest{
			StationID:    "station-1",
			LanguageCode: "en",
			Title:        "", // Empty title
			Description:  "Test description",
		}

		mockStationRepo.On("FindByID", "station-1").Return(station, nil)

		// Act
		result, err := service.CreateTranslation(req)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrInvalidTitle, err)
		mockStationRepo.AssertExpectations(t)
	})
}

func TestTranslationService_UpdateTranslation(t *testing.T) {
	t.Run("success - update existing translation", func(t *testing.T) {
		// Arrange
		mockTranslationRepo := new(MockTranslationRepository)
		mockStationRepo := new(MockStationRepository)
		service := NewTranslationService(mockTranslationRepo, mockStationRepo)

		existing := &domain.StationTranslation{
			StationID:    "station-1",
			LanguageCode: i18n.LanguageEN,
			Title:        "Old Title",
			Description:  "Old Description",
			Keywords:     []string{"old"},
		}

		req := &domain.UpdateTranslationRequest{
			Title:       "New Title",
			Description: "New Description",
			Keywords:    []string{"new", "updated"},
		}

		mockTranslationRepo.On("Get", "station-1", i18n.LanguageEN).Return(existing, nil)
		mockTranslationRepo.On("Update", mock.AnythingOfType("*domain.StationTranslation")).Return(nil)

		// Act
		result, err := service.UpdateTranslation("station-1", i18n.LanguageEN, req)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "New Title", result.Title)
		assert.Equal(t, "New Description", result.Description)
		mockTranslationRepo.AssertExpectations(t)
	})

	t.Run("error - translation not found", func(t *testing.T) {
		// Arrange
		mockTranslationRepo := new(MockTranslationRepository)
		mockStationRepo := new(MockStationRepository)
		service := NewTranslationService(mockTranslationRepo, mockStationRepo)

		req := &domain.UpdateTranslationRequest{
			Title:       "New Title",
			Description: "New Description",
		}

		mockTranslationRepo.On("Get", "station-1", i18n.LanguageEN).Return(nil, domain.ErrTranslationNotFound)

		// Act
		result, err := service.UpdateTranslation("station-1", i18n.LanguageEN, req)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrTranslationNotFound, err)
		mockTranslationRepo.AssertExpectations(t)
	})
}

func TestTranslationService_GetOrGenerateTranslation(t *testing.T) {
	t.Run("returns translation from database", func(t *testing.T) {
		// Arrange
		mockTranslationRepo := new(MockTranslationRepository)
		mockStationRepo := new(MockStationRepository)
		service := NewTranslationService(mockTranslationRepo, mockStationRepo)

		station := &domain.Station{
			ID:      "station-1",
			Name:    "Test Radio",
			Country: "Argentina",
		}

		dbTranslation := &domain.StationTranslation{
			StationID:    "station-1",
			LanguageCode: i18n.LanguageEN,
			Title:        "Professional Translation",
			Description:  "Professional description from database",
			Keywords:     []string{"professional", "radio"},
		}

		mockTranslationRepo.On("Get", "station-1", i18n.LanguageEN).Return(dbTranslation, nil)

		// Act
		result := service.GetOrGenerateTranslation(station, i18n.LanguageEN)

		// Assert
		assert.NotNil(t, result)
		assert.Equal(t, "Professional Translation", result.Title)
		assert.Equal(t, "Professional description from database", result.Description)
		mockTranslationRepo.AssertExpectations(t)
	})

	t.Run("generates default translation when not in database", func(t *testing.T) {
		// Arrange
		mockTranslationRepo := new(MockTranslationRepository)
		mockStationRepo := new(MockStationRepository)
		service := NewTranslationService(mockTranslationRepo, mockStationRepo)

		station := &domain.Station{
			ID:      "station-1",
			Name:    "Test Radio",
			Country: "Argentina",
			Tags:    []string{"rock"},
		}

		mockTranslationRepo.On("Get", "station-1", i18n.LanguageEN).Return(nil, errors.New("not found"))

		// Act
		result := service.GetOrGenerateTranslation(station, i18n.LanguageEN)

		// Assert
		assert.NotNil(t, result)
		assert.Equal(t, "station-1", result.StationID)
		assert.Equal(t, i18n.LanguageEN, result.LanguageCode)
		assert.Contains(t, result.Title, "Test Radio")
		assert.Contains(t, result.Description, "Test Radio")
		assert.Contains(t, result.Keywords, "Test Radio")
		mockTranslationRepo.AssertExpectations(t)
	})

	t.Run("generates translations in all supported languages", func(t *testing.T) {
		// Arrange
		mockTranslationRepo := new(MockTranslationRepository)
		mockStationRepo := new(MockStationRepository)
		service := NewTranslationService(mockTranslationRepo, mockStationRepo)

		station := &domain.Station{
			ID:      "station-1",
			Name:    "Test Radio",
			Country: "Argentina",
		}

		languages := []i18n.Language{i18n.LanguageES, i18n.LanguageEN, i18n.LanguageFR, i18n.LanguageDE}

		for _, lang := range languages {
			mockTranslationRepo.On("Get", "station-1", lang).Return(nil, errors.New("not found"))
		}

		// Act & Assert
		for _, lang := range languages {
			result := service.GetOrGenerateTranslation(station, lang)
			assert.NotNil(t, result)
			assert.Equal(t, lang, result.LanguageCode)
			assert.NotEmpty(t, result.Title)
			assert.NotEmpty(t, result.Description)
		}

		mockTranslationRepo.AssertExpectations(t)
	})
}

func TestTranslationService_BulkCreateTranslations(t *testing.T) {
	t.Run("success - bulk create valid translations", func(t *testing.T) {
		// Arrange
		mockTranslationRepo := new(MockTranslationRepository)
		mockStationRepo := new(MockStationRepository)
		service := NewTranslationService(mockTranslationRepo, mockStationRepo)

		translations := []*domain.StationTranslation{
			{
				StationID:    "station-1",
				LanguageCode: i18n.LanguageEN,
				Title:        "Title 1",
				Description:  "Description 1",
				Keywords:     []string{"test"},
			},
			{
				StationID:    "station-2",
				LanguageCode: i18n.LanguageES,
				Title:        "Título 2",
				Description:  "Descripción 2",
				Keywords:     []string{"prueba"},
			},
		}

		mockTranslationRepo.On("BulkCreate", translations).Return(nil)

		// Act
		err := service.BulkCreateTranslations(translations)

		// Assert
		assert.NoError(t, err)
		mockTranslationRepo.AssertExpectations(t)
	})

	t.Run("error - invalid translation in bulk", func(t *testing.T) {
		// Arrange
		mockTranslationRepo := new(MockTranslationRepository)
		mockStationRepo := new(MockStationRepository)
		service := NewTranslationService(mockTranslationRepo, mockStationRepo)

		translations := []*domain.StationTranslation{
			{
				StationID:    "station-1",
				LanguageCode: i18n.LanguageEN,
				Title:        "", // Invalid: empty title
				Description:  "Description 1",
			},
		}

		// Act
		err := service.BulkCreateTranslations(translations)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid translation")
	})
}
