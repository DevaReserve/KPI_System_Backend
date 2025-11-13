package db_var

import "KPI_System_Backend/models"

// Database table names
const (
	TableUsers                 = "users"
	TableEmployees             = "employees"
	TableDivisions             = "divisions"
	TableEvaluations           = "evaluations"
	TablePerformanceIndicators = "performance_indicators"
	TableEvaluationScores      = "evaluation_scores"
)

// User roles constants
const (
	RoleAdmin    = "admin"
	RoleManager  = "manager"
	RoleEmployee = "employee"
)

// Evaluation status constants
const (
	EvaluationStatusDraft     = "draft"
	EvaluationStatusSubmitted = "submitted"
	EvaluationStatusApproved  = "approved"
)

// Indicator types constants
const (
	IndicatorTypeUmum    = "umum"
	IndicatorTypeSpesifik = "spesifik"
)

// Score conversion map (Skor 1-5 ke Poin 1-100)
var ScoreConversion = map[int]int{
	1: 20, // Sangat Kurang
	2: 40, // Kurang
	3: 60, // Cukup
	4: 80, // Baik
	5: 100, // Sangat Baik
}

// Performance categories based on final score
var PerformanceCategories = map[string]struct {
	MinScore int
	MaxScore int
	Grade    string
	Label    string
}{
	"sangat_baik": {86, 100, "A", "Sangat Baik"},
	"baik":        {71, 85, "B", "Baik"},
	"cukup":       {56, 70, "C", "Cukup"},
	"kurang":      {41, 55, "D", "Kurang"},
	"sangat_kurang": {0, 40, "E", "Sangat Kurang"},
}

// Default indicators for quick setup
var DefaultIndicators = []models.PerformanceIndicator{
	// Umum indicators (for all divisions)
	{
		Name:          "Kehadiran dan Disiplin",
		Description:   "Ketepatan waktu dan tingkat kehadiran",
		IndicatorType: IndicatorTypeUmum,
		Weight:        15.0,
		DivisionID:    nil, // nil means applicable to all divisions
	},
	{
		Name:          "Sikap Kerja (Attitude)",
		Description:   "Inisiatif, etos kerja, dan profesionalisme",
		IndicatorType: IndicatorTypeUmum,
		Weight:        15.0,
		DivisionID:    nil,
	},
	{
		Name:          "Kerja Sama Tim",
		Description:   "Kemampuan berkolaborasi dengan rekan kerja",
		IndicatorType: IndicatorTypeUmum,
		Weight:        20.0,
		DivisionID:    nil,
	},
	{
		Name:          "Tanggung Jawab",
		Description:   "Kemampuan menyelesaikan tugas sesuai tenggat waktu dan standar",
		IndicatorType: IndicatorTypeUmum,
		Weight:        20.0,
		DivisionID:    nil,
	},
}

// Division-specific indicators template
var DivisionSpecificIndicators = map[string][]models.PerformanceIndicator{
	"Development": {
		{
			Name:          "Kualitas Kode",
			Description:   "Efisiensi, kebersihan, dan minimnya bug pada kode yang ditulis",
			IndicatorType: IndicatorTypeSpesifik,
			Weight:        20.0,
		},
		{
			Name:          "Ketepatan Waktu Proyek",
			Description:   "Kemampuan menyelesaikan task atau sprint sesuai jadwal",
			IndicatorType: IndicatorTypeSpesifik,
			Weight:        10.0,
		},
	},
	"Marketing": {
		{
			Name:          "Pencapaian Target",
			Description:   "Jumlah leads yang dihasilkan atau target penjualan yang tercapai",
			IndicatorType: IndicatorTypeSpesifik,
			Weight:        25.0,
		},
		{
			Name:          "Efektivitas Kampanye",
			Description:   "Tingkat keterlibatan (engagement rate) dari kampanye yang dijalankan",
			IndicatorType: IndicatorTypeSpesifik,
			Weight:        15.0,
		},
	},
	"IT Support": {
		{
			Name:          "Waktu Respon (Response Time)",
			Description:   "Kecepatan dalam menanggapi keluhan user",
			IndicatorType: IndicatorTypeSpesifik,
			Weight:        20.0,
		},
		{
			Name:          "Tingkat Penyelesaian (Resolution Rate)",
			Description:   "Persentase masalah yang berhasil diselesaikan",
			IndicatorType: IndicatorTypeSpesifik,
			Weight:        10.0,
		},
	},
}

// SQL queries for complex operations
const (
	// Query untuk mendapatkan evaluasi yang pending untuk manager
	QueryPendingEvaluations = `
		SELECT e.*, emp.name as employee_name, d.name as division_name
		FROM evaluations e
		JOIN employees emp ON e.employee_id = emp.id
		JOIN divisions d ON emp.division_id = d.id
		WHERE e.evaluator_id = ? AND e.status = 'draft'
		ORDER BY e.created_at DESC
	`

	// Query untuk mendapatkan performance summary per divisi
	QueryDivisionPerformance = `
		SELECT 
			d.name as division_name,
			COUNT(DISTINCT e.employee_id) as total_employees,
			AVG(e.total_score) as average_score,
			COUNT(CASE WHEN e.status = 'submitted' THEN 1 END) as completed_evaluations
		FROM divisions d
		LEFT JOIN employees emp ON d.id = emp.division_id
		LEFT JOIN evaluations e ON emp.id = e.employee_id 
			AND e.id = (SELECT MAX(id) FROM evaluations WHERE employee_id = emp.id AND status = 'submitted')
		GROUP BY d.id, d.name
		ORDER BY average_score DESC
	`

	// Query untuk mendapatkan trend kinerja employee
	QueryEmployeePerformanceTrend = `
		SELECT 
			ep.name as period_name,
			e.total_score,
			e.submitted_at
		FROM evaluations e
		JOIN evaluation_periods ep ON e.period_id = ep.id
		WHERE e.employee_id = ? AND e.status = 'submitted'
		ORDER BY e.submitted_at ASC
	`

	// Query untuk dashboard manager
	QueryManagerDashboard = `
		SELECT 
			COUNT(DISTINCT emp.id) as total_team_members,
			COUNT(DISTINCT CASE WHEN e.status = 'submitted' THEN e.employee_id END) as evaluated_members,
			AVG(CASE WHEN e.status = 'submitted' THEN e.total_score END) as team_average_score,
			COUNT(DISTINCT CASE WHEN e.status = 'draft' THEN e.employee_id END) as pending_evaluations
		FROM employees emp
		LEFT JOIN evaluations e ON emp.id = e.employee_id 
			AND e.period_id = ?
		WHERE emp.direct_supervisor_id = ? OR emp.division_id IN (
			SELECT id FROM divisions WHERE manager_id = ?
		)
	`
)

// Validation messages
const (
	ErrWeightNot100         = "Total bobot indikator harus 100%"
	ErrDuplicateNIP         = "NIP sudah terdaftar"
	ErrDuplicateEmail       = "Email sudah terdaftar"
	ErrDuplicateUsername    = "Username sudah terdaftar"
	ErrEvaluationExists     = "Evaluasi untuk periode ini sudah ada"
	ErrInvalidScore         = "Nilai harus antara 1-5"
	ErrManagerNoAccess      = "Manager tidak memiliki akses ke karyawan ini"
	ErrDivisionHasEmployees = "Divisi tidak dapat dihapus karena masih memiliki karyawan"
)

// Success messages
const (
	MsgLoginSuccess       = "Login berhasil"
	MsgEvaluationSaved    = "Evaluasi berhasil disimpan"
	MsgEvaluationSubmitted = "Evaluasi berhasil disubmit"
	MsgEmployeeCreated    = "Karyawan berhasil dibuat"
	MsgEmployeeUpdated    = "Karyawan berhasil diperbarui"
	MsgIndicatorCreated   = "Indikator berhasil dibuat"
	MsgDivisionCreated    = "Divisi berhasil dibuat"
)

// Pagination defaults
const (
	DefaultPage     = 1
	DefaultPageSize = 10
	MaxPageSize     = 100
)

// Cache keys pattern
const (
	CacheKeyUserProfile    = "user_profile:%d"
	CacheKeyEmployee       = "employee:%d"
	CacheKeyDivision       = "division:%d"
	CacheKeyIndicators     = "indicators:%d" // division_id
	CacheKeyEvaluation     = "evaluation:%d"
	CacheKeyDashboardAdmin = "dashboard_admin"
	CacheKeyDashboardManager = "dashboard_manager:%d" // manager_id
)