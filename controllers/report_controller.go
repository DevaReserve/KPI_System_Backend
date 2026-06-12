package controllers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

type ReportController struct {
    DB *gorm.DB
}

func NewReportController(db *gorm.DB) *ReportController {
    return &ReportController{DB: db}
}

// Struct untuk hasil report
type EvaluationReport struct {
    EmployeeName string  `json:"employee_name"`
    
    // PERBAIKAN 1: Tambahkan tag gorm:"column:..." agar mapping akurat
    NIP          string  `json:"nip" gorm:"column:nip_pegawai"` 
    
    Division     string  `json:"division"`
    Position     string  `json:"position"`
    Evaluator    string  `json:"evaluator"`
    TotalScore   float64 `json:"total_score"`
    Grade        string  `json:"grade"` 
    Feedback     string  `json:"feedback"`
}

func (rc *ReportController) GetEvaluationReport(c *gin.Context) {
    periodID := c.Query("period_id")

    if periodID == "" {
        Response(c, http.StatusBadRequest, "Parameter period_id wajib diisi", nil)
        return
    }

    // Inisialisasi slice kosong (bukan nil) agar frontend tidak error .length
    results := make([]EvaluationReport, 0)

    // Query Join Kompleks
    query := `
        SELECT 
            e.name as employee_name,
            
            -- PERBAIKAN 2: Ubah alias agar UNIK dan SAMA PERSIS dengan tag gorm
            e.n_ip as nip_pegawai, 
            
            d.name as division,
            e.position as position,
            sup.name as evaluator,
            
            -- Gunakan COALESCE untuk mencegah error jika nilai NULL
            COALESCE(ev.total_score, 0) as total_score,
            ev.feedback as feedback
        FROM evaluations ev
        JOIN employees e ON ev.employee_id = e.id
        LEFT JOIN divisions d ON e.division_id = d.id
        LEFT JOIN employees sup ON ev.evaluator_id = sup.id
        WHERE ev.period_id = ? AND ev.status = 'submitted'
        ORDER BY d.name, e.name
    `

    if err := rc.DB.Raw(query, periodID).Scan(&results).Error; err != nil {
        Response(c, http.StatusInternalServerError, "Gagal mengambil data laporan", nil)
        return
    }

    // Hitung Grade manual
    for i, res := range results {
        if res.TotalScore >= 86 {
            results[i].Grade = "A"
        } else if res.TotalScore >= 71 {
            results[i].Grade = "B"
        } else if res.TotalScore >= 56 {
            results[i].Grade = "C"
        } else if res.TotalScore >= 41 {
            results[i].Grade = "D"
        } else {
            results[i].Grade = "E"
        }
    }

    Response(c, http.StatusOK, "Laporan berhasil dibuat", results)
}

// ============================================================
// GET /api/admin/reports/comparison?period_a=X&period_b=Y
// Membandingkan performa pegawai di 2 periode berbeda
// ============================================================
func (rc *ReportController) GetPeriodComparison(c *gin.Context) {
    periodA := c.Query("period_a")
    periodB := c.Query("period_b")

    if periodA == "" || periodB == "" {
        Response(c, http.StatusBadRequest, "Parameter period_a dan period_b wajib diisi", nil)
        return
    }

    type PeriodScore struct {
        EmployeeName string  `json:"employee_name"`
        NIP          string  `json:"nip"`
        Division     string  `json:"division"`
        Position     string  `json:"position"`
        ScoreA       float64 `json:"score_a"`
        ScoreB       float64 `json:"score_b"`
        Delta        float64 `json:"delta"`        // ScoreB - ScoreA
        Trend        string  `json:"trend"`         // "naik", "turun", "tetap", "baru"
        GradeA       string  `json:"grade_a"`
        GradeB       string  `json:"grade_b"`
    }

    // Ambil data periode A
    type RawScore struct {
        EmployeeName string
        NIP          string
        Division     string
        Position     string
        TotalScore   float64
    }

    getScores := func(periodID string) map[string]RawScore {
        var rows []struct {
            EmployeeName string  `gorm:"column:employee_name"`
            NIP          string  `gorm:"column:nip_pegawai"`
            Division     string  `gorm:"column:division"`
            Position     string  `gorm:"column:position"`
            TotalScore   float64 `gorm:"column:total_score"`
        }
        rc.DB.Raw(`
            SELECT e.name as employee_name, e.n_ip as nip_pegawai,
                d.name as division, e.position as position,
                COALESCE(ev.total_score, 0) as total_score
            FROM evaluations ev
            JOIN employees e ON ev.employee_id = e.id
            LEFT JOIN divisions d ON e.division_id = d.id
            WHERE ev.period_id = ? AND ev.status = 'submitted'
        `, periodID).Scan(&rows)

        m := make(map[string]RawScore)
        for _, r := range rows {
            m[r.NIP] = RawScore{r.EmployeeName, r.NIP, r.Division, r.Position, r.TotalScore}
        }
        return m
    }

    gradeOf := func(s float64) string {
        if s >= 86 { return "A" }
        if s >= 71 { return "B" }
        if s >= 56 { return "C" }
        if s >= 41 { return "D" }
        return "E"
    }

    scoresA := getScores(periodA)
    scoresB := getScores(periodB)

    // Gabungkan semua pegawai dari kedua periode
    nipSet := make(map[string]bool)
    for nip := range scoresA { nipSet[nip] = true }
    for nip := range scoresB { nipSet[nip] = true }

    var results []PeriodScore
    for nip := range nipSet {
        a, hasA := scoresA[nip]
        b, hasB := scoresB[nip]

        var row PeriodScore
        if hasA {
            row.EmployeeName = a.EmployeeName
            row.NIP = a.NIP
            row.Division = a.Division
            row.Position = a.Position
        } else {
            row.EmployeeName = b.EmployeeName
            row.NIP = b.NIP
            row.Division = b.Division
            row.Position = b.Position
        }

        row.ScoreA = 0
        row.ScoreB = 0
        row.GradeA = "-"
        row.GradeB = "-"

        if hasA {
            row.ScoreA = a.TotalScore
            row.GradeA = gradeOf(a.TotalScore)
        }
        if hasB {
            row.ScoreB = b.TotalScore
            row.GradeB = gradeOf(b.TotalScore)
        }

        row.Delta = row.ScoreB - row.ScoreA

        switch {
        case !hasA && hasB:
            row.Trend = "baru"
        case row.Delta > 0.5:
            row.Trend = "naik"
        case row.Delta < -0.5:
            row.Trend = "turun"
        default:
            row.Trend = "tetap"
        }

        results = append(results, row)
    }

    if results == nil {
        results = []PeriodScore{}
    }

    Response(c, http.StatusOK, "Komparasi periode berhasil dibuat", results)
}