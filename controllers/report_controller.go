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