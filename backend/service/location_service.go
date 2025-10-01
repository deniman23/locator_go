package service

import (
	"fmt"
	"locator/dao"
	"locator/models"
	"log"
	"math"
	"sort"
	"sync"
	"time"
)

// LocationService отвечает за бизнес-логику, связанную с операциями над местоположениями.
type LocationService struct {
	DAO *dao.LocationDAO

	// Кэш для значимых локаций
	cache      map[string]*cacheEntry
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
}

// cacheEntry представляет запись в кэше
type cacheEntry struct {
	data      []models.Location
	timestamp time.Time
}

// NewLocationService создаёт новый экземпляр сервиса.
func NewLocationService(dao *dao.LocationDAO) *LocationService {
	return &LocationService{
		DAO:      dao,
		cache:    make(map[string]*cacheEntry),
		cacheTTL: 5 * time.Minute, // Кэш на 5 минут (соответствует интервалу обновления)
	}
}

// GetLocation получает данные о местоположении для заданного пользователя.
func (svc *LocationService) GetLocation(userID int) (*models.Location, error) {
	log.Printf("[GetLocation] Запрос на получение местоположения для userID=%d", userID)
	location, err := svc.DAO.GetByUserID(userID)
	if err != nil {
		log.Printf("[GetLocation] Ошибка при получении местоположения для userID=%d: %v", userID, err)
		return nil, err
	}
	log.Printf("[GetLocation] Запись о местоположении получена для userID=%d: Latitude=%.6f, Longitude=%.6f", userID, location.Latitude, location.Longitude)
	return location, nil
}

// CreateLocation создаёт новую запись о местоположении без обновления существующей.
func (svc *LocationService) CreateLocation(userID int, lat, lon float64) (*models.Location, error) {
	log.Printf("[CreateLocation] Создание записи местоположения: userID=%d, Latitude=%.6f, Longitude=%.6f", userID, lat, lon)
	newLocation := models.NewLocation(userID, lat, lon)
	if err := svc.DAO.Create(newLocation); err != nil {
		log.Printf("[CreateLocation] Ошибка при создании записи для userID=%d: %v", userID, err)
		return nil, err
	}

	// Инвалидируем кэш при добавлении новой локации
	svc.invalidateCache()

	log.Printf("[CreateLocation] Запись успешно создана для userID=%d: Latitude=%.6f, Longitude=%.6f", userID, lat, lon)
	return newLocation, nil
}

// GetLocations возвращает только значимые локации для отображения на карте
func (svc *LocationService) GetLocations() ([]models.Location, error) {
	return svc.GetLocationsWithCache(false)
}

// GetLocationsWithCache возвращает значимые локации с возможностью принудительного обновления кэша
func (svc *LocationService) GetLocationsWithCache(forceRefresh bool) ([]models.Location, error) {
	log.Printf("[GetLocations] Запрос на получение значимых локаций (forceRefresh=%v)", forceRefresh)

	cacheKey := "all_locations"

	// Проверяем кэш, если не требуется принудительное обновление
	if !forceRefresh {
		if cached := svc.getFromCache(cacheKey); cached != nil {
			log.Printf("[GetLocations] Возвращаем %d локаций из кэша", len(cached))
			return cached, nil
		}
	}

	// Получаем все локации из БД
	allLocations, err := svc.DAO.GetAll()
	if err != nil {
		log.Printf("[GetLocations] Ошибка при получении записей: %v", err)
		return nil, err
	}

	log.Printf("[GetLocations] Получено %d записей из БД, начинаем фильтрацию", len(allLocations))

	// Фильтруем и возвращаем только значимые точки
	significantLocations := svc.filterSignificantLocations(allLocations)

	// Сохраняем в кэш
	svc.saveToCache(cacheKey, significantLocations)

	log.Printf("[GetLocations] Отфильтровано %d значимых локаций из %d общих (сохранено в кэш)",
		len(significantLocations), len(allLocations))
	return significantLocations, nil
}

// GetLocationsBetween возвращает значимые локации за указанный период
func (svc *LocationService) GetLocationsBetween(fromStr, toStr string) ([]models.Location, error) {
	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		return nil, fmt.Errorf("неверный формат параметра 'from': %v", err)
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		return nil, fmt.Errorf("неверный формат параметра 'to': %v", err)
	}
	if from.After(to) {
		return nil, fmt.Errorf("начало интервала не может быть позже окончания")
	}

	// Формируем ключ кэша для временного диапазона
	cacheKey := fmt.Sprintf("locations_%s_%s", fromStr, toStr)

	// Проверяем кэш
	if cached := svc.getFromCache(cacheKey); cached != nil {
		log.Printf("[GetLocationsBetween] Возвращаем %d локаций из кэша для периода %s - %s",
			len(cached), fromStr, toStr)
		return cached, nil
	}

	// Получаем локации за период
	allLocations, err := svc.DAO.GetLocationsBetween(from, to)
	if err != nil {
		return nil, err
	}

	// Фильтруем значимые локации
	significantLocations := svc.filterSignificantLocations(allLocations)

	// Сохраняем в кэш
	svc.saveToCache(cacheKey, significantLocations)

	log.Printf("[GetLocationsBetween] Отфильтровано %d значимых локаций для периода %s - %s (сохранено в кэш)",
		len(significantLocations), fromStr, toStr)

	return significantLocations, nil
}

// getFromCache получает данные из кэша если они актуальны
func (svc *LocationService) getFromCache(key string) []models.Location {
	svc.cacheMutex.RLock()
	defer svc.cacheMutex.RUnlock()

	entry, exists := svc.cache[key]
	if !exists {
		return nil
	}

	// Проверяем TTL
	if time.Since(entry.timestamp) > svc.cacheTTL {
		return nil
	}

	// Возвращаем копию данных
	result := make([]models.Location, len(entry.data))
	copy(result, entry.data)
	return result
}

// saveToCache сохраняет данные в кэш
func (svc *LocationService) saveToCache(key string, data []models.Location) {
	svc.cacheMutex.Lock()
	defer svc.cacheMutex.Unlock()

	// Создаем копию данных для кэша
	dataCopy := make([]models.Location, len(data))
	copy(dataCopy, data)

	svc.cache[key] = &cacheEntry{
		data:      dataCopy,
		timestamp: time.Now(),
	}
}

// invalidateCache очищает весь кэш
func (svc *LocationService) invalidateCache() {
	svc.cacheMutex.Lock()
	defer svc.cacheMutex.Unlock()

	svc.cache = make(map[string]*cacheEntry)
	log.Println("[invalidateCache] Кэш очищен")
}

// clearOldCache удаляет устаревшие записи из кэша (можно вызывать периодически)
func (svc *LocationService) clearOldCache() {
	svc.cacheMutex.Lock()
	defer svc.cacheMutex.Unlock()

	now := time.Now()
	for key, entry := range svc.cache {
		if now.Sub(entry.timestamp) > svc.cacheTTL {
			delete(svc.cache, key)
			log.Printf("[clearOldCache] Удалена устаревшая запись кэша: %s", key)
		}
	}
}

// SetCacheTTL позволяет изменить время жизни кэша
func (svc *LocationService) SetCacheTTL(ttl time.Duration) {
	svc.cacheTTL = ttl
	svc.invalidateCache()
	log.Printf("[SetCacheTTL] TTL кэша изменен на %v", ttl)
}

// filterSignificantLocations фильтрует только значимые локации из всех
func (svc *LocationService) filterSignificantLocations(allLocations []models.Location) []models.Location {
	// Параметры фильтрации
	const (
		maxDistance = 100.0            // макс расстояние между точками в кластере (метры)
		minDuration = 15 * time.Minute // мин время нахождения в одном месте
		minPoints   = 3                // минимум 3 точки (15 минут при интервале 5 мин)
	)

	// Группируем по пользователям
	userLocations := make(map[int][]models.Location)
	for _, loc := range allLocations {
		userLocations[loc.UserID] = append(userLocations[loc.UserID], loc)
	}

	var significantLocations []models.Location

	for userID, locations := range userLocations {
		sort.Slice(locations, func(i, j int) bool {
			return locations[i].CreatedAt.Before(locations[j].CreatedAt)
		})

		// Пытаемся найти кластеры
		clusters := svc.clusterLocations(locations, maxDistance, minDuration, minPoints)

		if len(clusters) > 0 {
			// Если есть кластеры - добавляем их
			significantLocations = append(significantLocations, clusters...)
		} else {
			// Если кластеров нет - добавляем репрезентативные точки
			// чтобы пользователь не исчез с карты
			representativePoints := svc.getRepresentativePoints(locations)
			significantLocations = append(significantLocations, representativePoints...)

			log.Printf("[filterSignificantLocations] Пользователь %d не имеет кластеров, добавлено %d репрезентативных точек",
				userID, len(representativePoints))
		}
	}

	return significantLocations
}

// getRepresentativePoints возвращает важные точки для пользователя без кластеров
func (svc *LocationService) getRepresentativePoints(locations []models.Location) []models.Location {
	if len(locations) == 0 {
		return nil
	}

	if len(locations) <= 3 {
		return locations // Если мало точек, возвращаем все
	}

	const (
		minDistanceMeters = 500.0            // Минимальное расстояние для значимого перемещения (метры)
		minTimeDiff       = 10 * time.Minute // Минимальная разница во времени
		maxPoints         = 10               // Максимум точек на пользователя
	)

	var result []models.Location

	// Всегда добавляем первую точку
	result = append(result, locations[0])

	for i := 1; i < len(locations); i++ {
		lastAdded := result[len(result)-1]
		current := locations[i]

		// Расчет расстояния от последней добавленной точки
		distance := svc.haversineDistance(
			lastAdded.Latitude, lastAdded.Longitude,
			current.Latitude, current.Longitude,
		)

		// Расчет времени между точками
		timeDiff := current.CreatedAt.Sub(lastAdded.CreatedAt)

		// Условия для добавления точки:
		// 1. Значительное перемещение (более 500 метров)
		// 2. Или прошло достаточно времени (более 10 минут)
		if distance >= minDistanceMeters || timeDiff >= minTimeDiff {
			result = append(result, current)

			// Ограничиваем количество точек
			if len(result) >= maxPoints {
				break
			}
		}
	}

	// Всегда добавляем последнюю точку (если её ещё нет)
	lastLocation := locations[len(locations)-1]
	if len(result) < maxPoints && result[len(result)-1].ID != lastLocation.ID {
		// Проверяем, не слишком ли близко последняя точка
		lastAdded := result[len(result)-1]
		distance := svc.haversineDistance(
			lastAdded.Latitude, lastAdded.Longitude,
			lastLocation.Latitude, lastLocation.Longitude,
		)
		// Добавляем если расстояние больше 100 метров
		if distance >= 100 {
			result = append(result, lastLocation)
		}
	}

	return result
}

// clusterLocations группирует близкие точки и возвращает представительные локации
func (svc *LocationService) clusterLocations(
	locations []models.Location,
	maxDistance float64,
	minDuration time.Duration,
	minPoints int,
) []models.Location {
	if len(locations) == 0 {
		return nil
	}

	var result []models.Location
	var currentCluster []models.Location

	for _, loc := range locations {
		if len(currentCluster) == 0 {
			currentCluster = append(currentCluster, loc)
			continue
		}

		// Вычисляем центр текущего кластера
		centerLat, centerLon := svc.calculateCenter(currentCluster)
		distance := svc.haversineDistance(loc.Latitude, loc.Longitude, centerLat, centerLon)

		// Проверяем временной разрыв (если больше часа - новый кластер)
		lastTime := currentCluster[len(currentCluster)-1].CreatedAt
		timeDiff := loc.CreatedAt.Sub(lastTime)

		if distance <= maxDistance && timeDiff <= time.Hour {
			// Добавляем в текущий кластер
			currentCluster = append(currentCluster, loc)
		} else {
			// Сохраняем текущий кластер если он значимый
			if representative := svc.getRepresentativeLocation(currentCluster, minDuration, minPoints); representative != nil {
				result = append(result, *representative)
			}
			// Начинаем новый кластер
			currentCluster = []models.Location{loc}
		}
	}

	// Обрабатываем последний кластер
	if representative := svc.getRepresentativeLocation(currentCluster, minDuration, minPoints); representative != nil {
		result = append(result, *representative)
	}

	return result
}

// getRepresentativeLocation возвращает одну представительную точку для кластера
func (svc *LocationService) getRepresentativeLocation(
	cluster []models.Location,
	minDuration time.Duration,
	minPoints int,
) *models.Location {
	// Проверяем минимальное количество точек
	if len(cluster) < minPoints {
		return nil
	}

	// Проверяем длительность нахождения
	firstTime := cluster[0].CreatedAt
	lastTime := cluster[len(cluster)-1].CreatedAt
	duration := lastTime.Sub(firstTime)

	if duration < minDuration {
		return nil
	}

	// Вычисляем центр кластера
	centerLat, centerLon := svc.calculateCenter(cluster)

	// Возвращаем представительную точку с усредненными координатами
	return &models.Location{
		UserID:    cluster[0].UserID,
		Latitude:  centerLat,
		Longitude: centerLon,
		CreatedAt: lastTime,
		UpdatedAt: lastTime,
	}
}

// calculateCenter вычисляет географический центр группы локаций
func (svc *LocationService) calculateCenter(locations []models.Location) (float64, float64) {
	var sumLat, sumLon float64
	for _, loc := range locations {
		sumLat += loc.Latitude
		sumLon += loc.Longitude
	}
	n := float64(len(locations))
	return sumLat / n, sumLon / n
}

// haversineDistance вычисляет расстояние между двумя точками в метрах
func (svc *LocationService) haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // Радиус Земли в метрах

	// Перевод градусов в радианы
	rLat1 := lat1 * math.Pi / 180
	rLat2 := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(rLat1)*math.Cos(rLat2)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
