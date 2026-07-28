package controllers

import (
    "fmt"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/xuri/excelize/v2"
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

// ============================================================
// GET /api/admin/reports/evaluations/export-excel?period_id=X
// Export laporan evaluasi ke file Excel (.xlsx)
// ============================================================
func (rc *ReportController) ExportEvaluationReportExcel(c *gin.Context) {
    periodID := c.Query("period_id")
    if periodID == "" {
        Response(c, http.StatusBadRequest, "Parameter period_id wajib diisi", nil)
        return
    }

    results := make([]EvaluationReport, 0)
    query := `
        SELECT
            e.name as employee_name,
            e.n_ip as nip_pegawai,
            d.name as division,
            e.position as position,
            sup.name as evaluator,
            COALESCE(ev.total_score, 0) as total_score,
            ev.feedback as feedback
        FROM evaluations ev
        JOIN employees e ON ev.employee_id = e.id
        LEFT JOIN divisions d ON e.division_id = d.id
        LEFT JOIN employees sup ON ev.evaluator_id = sup.id
        WHERE ev.period_id = ? AND ev.status = 'submitted'
        ORDER BY d.name, ev.total_score DESC
    `
    if err := rc.DB.Raw(query, periodID).Scan(&results).Error; err != nil {
        Response(c, http.StatusInternalServerError, "Gagal mengambil data laporan", nil)
        return
    }

    gradeOf := func(s float64) string {
        if s >= 86 { return "A" }
        if s >= 71 { return "B" }
        if s >= 56 { return "C" }
        if s >= 41 { return "D" }
        return "E"
    }
    for i, res := range results {
        results[i].Grade = gradeOf(res.TotalScore)
    }

    // Ambil nama periode dari DB
    var periodName string
    rc.DB.Raw("SELECT name FROM evaluation_periods WHERE id = ?", periodID).Scan(&periodName)
    if periodName == "" {
        periodName = "Periode"
    }

    f := excelize.NewFile()
    defer f.Close()

    sheet := "Laporan KPI"
    f.SetSheetName("Sheet1", sheet)

    // --- Style Definitions ---
    titleStyle, _ := f.NewStyle(&excelize.Style{
        Font:      &excelize.Font{Bold: true, Size: 14, Color: "FFFFFF"},
        Fill:      excelize.Fill{Type: "pattern", Color: []string{"2C3E50"}, Pattern: 1},
        Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
    })
    headerStyle, _ := f.NewStyle(&excelize.Style{
        Font:      &excelize.Font{Bold: true, Size: 10, Color: "FFFFFF"},
        Fill:      excelize.Fill{Type: "pattern", Color: []string{"2980B9"}, Pattern: 1},
        Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
        Border: []excelize.Border{
            {Type: "left", Color: "FFFFFF", Style: 1},
            {Type: "right", Color: "FFFFFF", Style: 1},
        },
    })
    evenRowStyle, _ := f.NewStyle(&excelize.Style{
        Fill:      excelize.Fill{Type: "pattern", Color: []string{"EBF5FB"}, Pattern: 1},
        Alignment: &excelize.Alignment{Vertical: "center"},
        Border: []excelize.Border{
            {Type: "left", Color: "D5D8DC", Style: 1},
            {Type: "right", Color: "D5D8DC", Style: 1},
            {Type: "bottom", Color: "D5D8DC", Style: 1},
        },
    })
    oddRowStyle, _ := f.NewStyle(&excelize.Style{
        Fill:      excelize.Fill{Type: "pattern", Color: []string{"FFFFFF"}, Pattern: 1},
        Alignment: &excelize.Alignment{Vertical: "center"},
        Border: []excelize.Border{
            {Type: "left", Color: "D5D8DC", Style: 1},
            {Type: "right", Color: "D5D8DC", Style: 1},
            {Type: "bottom", Color: "D5D8DC", Style: 1},
        },
    })
    centerStyle, _ := f.NewStyle(&excelize.Style{
        Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
        Border: []excelize.Border{
            {Type: "left", Color: "D5D8DC", Style: 1},
            {Type: "right", Color: "D5D8DC", Style: 1},
            {Type: "bottom", Color: "D5D8DC", Style: 1},
        },
    })
    gradeStyles := map[string]int{}
    gradeColors := map[string]string{"A": "1ABC9C", "B": "3498DB", "C": "F39C12", "D": "E67E22", "E": "E74C3C"}
    for g, color := range gradeColors {
        s, _ := f.NewStyle(&excelize.Style{
            Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
            Fill:      excelize.Fill{Type: "pattern", Color: []string{color}, Pattern: 1},
            Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
        })
        gradeStyles[g] = s
    }

    // --- Title Row ---
    f.MergeCell(sheet, "A1", "H1")
    f.SetCellValue(sheet, "A1", "LAPORAN REKAPITULASI PENILAIAN KINERJA")
    f.SetCellStyle(sheet, "A1", "H1", titleStyle)
    f.SetRowHeight(sheet, 1, 30)

    f.MergeCell(sheet, "A2", "H2")
    f.SetCellValue(sheet, "A2", fmt.Sprintf("Periode: %s | Dicetak: %s", periodName, time.Now().Format("02 January 2006")))
    subTitleStyle, _ := f.NewStyle(&excelize.Style{
        Font:      &excelize.Font{Italic: true, Size: 9, Color: "7F8C8D"},
        Alignment: &excelize.Alignment{Horizontal: "center"},
    })
    f.SetCellStyle(sheet, "A2", "H2", subTitleStyle)
    f.SetRowHeight(sheet, 2, 18)

    // --- Header Row (Row 3) ---
    headers := []string{"No", "NIP", "Nama Pegawai", "Divisi", "Jabatan", "Penilai", "Skor", "Grade"}
    for i, h := range headers {
        cell, _ := excelize.CoordinatesToCellName(i+1, 3)
        f.SetCellValue(sheet, cell, h)
        f.SetCellStyle(sheet, cell, cell, headerStyle)
    }
    f.SetRowHeight(sheet, 3, 22)

    // --- Data Rows ---
    for i, row := range results {
        rowNum := i + 4
        baseStyle := oddRowStyle
        if i%2 == 1 {
            baseStyle = evenRowStyle
        }
        f.SetCellValue(sheet, fmt.Sprintf("A%d", rowNum), i+1)
        f.SetCellValue(sheet, fmt.Sprintf("B%d", rowNum), row.NIP)
        f.SetCellValue(sheet, fmt.Sprintf("C%d", rowNum), row.EmployeeName)
        f.SetCellValue(sheet, fmt.Sprintf("D%d", rowNum), row.Division)
        f.SetCellValue(sheet, fmt.Sprintf("E%d", rowNum), row.Position)
        f.SetCellValue(sheet, fmt.Sprintf("F%d", rowNum), row.Evaluator)
        f.SetCellValue(sheet, fmt.Sprintf("G%d", rowNum), row.TotalScore)
        f.SetCellValue(sheet, fmt.Sprintf("H%d", rowNum), row.Grade)

        // Apply styles
        f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowNum), fmt.Sprintf("A%d", rowNum), centerStyle)
        f.SetCellStyle(sheet, fmt.Sprintf("B%d", rowNum), fmt.Sprintf("G%d", rowNum), baseStyle)
        if gs, ok := gradeStyles[row.Grade]; ok {
            f.SetCellStyle(sheet, fmt.Sprintf("H%d", rowNum), fmt.Sprintf("H%d", rowNum), gs)
        }
        f.SetRowHeight(sheet, rowNum, 18)
    }

    // --- Column Widths ---
    f.SetColWidth(sheet, "A", "A", 6)
    f.SetColWidth(sheet, "B", "B", 16)
    f.SetColWidth(sheet, "C", "C", 28)
    f.SetColWidth(sheet, "D", "D", 20)
    f.SetColWidth(sheet, "E", "E", 20)
    f.SetColWidth(sheet, "F", "F", 20)
    f.SetColWidth(sheet, "G", "G", 10)
    f.SetColWidth(sheet, "H", "H", 8)

    filename := fmt.Sprintf("Laporan_KPI_%s.xlsx", time.Now().Format("20060102"))
    c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
    c.Header("Cache-Control", "no-cache")

    if err := f.Write(c.Writer); err != nil {
        Response(c, http.StatusInternalServerError, "Gagal menulis file Excel", nil)
    }
}

// ============================================================
// GET /api/admin/reports/comparison/export-excel?period_a=X&period_b=Y
// Export laporan komparasi ke file Excel (.xlsx)
// ============================================================
func (rc *ReportController) ExportComparisonExcel(c *gin.Context) {
    periodA := c.Query("period_a")
    periodB := c.Query("period_b")
    if periodA == "" || periodB == "" {
        Response(c, http.StatusBadRequest, "Parameter period_a dan period_b wajib diisi", nil)
        return
    }

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
    nipSet := make(map[string]bool)
    for nip := range scoresA { nipSet[nip] = true }
    for nip := range scoresB { nipSet[nip] = true }

    type CompRow struct{ Name, NIP, Division, Position, GradeA, GradeB, Trend string; ScoreA, ScoreB, Delta float64 }
    var rows []CompRow
    for nip := range nipSet {
        a, hasA := scoresA[nip]; b, hasB := scoresB[nip]
        var row CompRow
        if hasA { row.Name = a.EmployeeName; row.NIP = a.NIP; row.Division = a.Division; row.Position = a.Position
        } else { row.Name = b.EmployeeName; row.NIP = b.NIP; row.Division = b.Division; row.Position = b.Position }
        row.GradeA = "-"; row.GradeB = "-"
        if hasA { row.ScoreA = a.TotalScore; row.GradeA = gradeOf(a.TotalScore) }
        if hasB { row.ScoreB = b.TotalScore; row.GradeB = gradeOf(b.TotalScore) }
        row.Delta = row.ScoreB - row.ScoreA
        switch {
        case !hasA && hasB: row.Trend = "Baru"
        case row.Delta > 0.5: row.Trend = "Naik"
        case row.Delta < -0.5: row.Trend = "Turun"
        default: row.Trend = "Tetap"
        }
        rows = append(rows, row)
    }

    var nameA, nameB string
    rc.DB.Raw("SELECT name FROM evaluation_periods WHERE id = ?", periodA).Scan(&nameA)
    rc.DB.Raw("SELECT name FROM evaluation_periods WHERE id = ?", periodB).Scan(&nameB)
    if nameA == "" { nameA = "Periode A" }
    if nameB == "" { nameB = "Periode B" }

    f := excelize.NewFile()
    defer f.Close()
    sheet := "Komparasi KPI"
    f.SetSheetName("Sheet1", sheet)

    titleStyle, _ := f.NewStyle(&excelize.Style{
        Font:      &excelize.Font{Bold: true, Size: 13, Color: "FFFFFF"},
        Fill:      excelize.Fill{Type: "pattern", Color: []string{"1A252F"}, Pattern: 1},
        Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
    })
    headerStyle, _ := f.NewStyle(&excelize.Style{
        Font:      &excelize.Font{Bold: true, Size: 9, Color: "FFFFFF"},
        Fill:      excelize.Fill{Type: "pattern", Color: []string{"2471A3"}, Pattern: 1},
        Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
    })
    baseStyle, _ := f.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{Vertical: "center"}, Border: []excelize.Border{{Type: "bottom", Color: "D5D8DC", Style: 1}}})
    centerStyle, _ := f.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}, Border: []excelize.Border{{Type: "bottom", Color: "D5D8DC", Style: 1}}})

    trendStyles := map[string]int{}
    trendColors := map[string]string{"Naik": "1ABC9C", "Turun": "E74C3C", "Tetap": "95A5A6", "Baru": "3498DB"}
    for t, col := range trendColors {
        s, _ := f.NewStyle(&excelize.Style{
            Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
            Fill: excelize.Fill{Type: "pattern", Color: []string{col}, Pattern: 1},
            Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
        })
        trendStyles[t] = s
    }

    f.MergeCell(sheet, "A1", "J1")
    f.SetCellValue(sheet, "A1", "LAPORAN KOMPARASI KINERJA ANTAR PERIODE")
    f.SetCellStyle(sheet, "A1", "J1", titleStyle)
    f.SetRowHeight(sheet, 1, 28)

    f.MergeCell(sheet, "A2", "J2")
    subStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Italic: true, Size: 9, Color: "7F8C8D"}, Alignment: &excelize.Alignment{Horizontal: "center"}})
    f.SetCellValue(sheet, "A2", fmt.Sprintf("Periode A: %s  |  Periode B: %s  |  Dicetak: %s", nameA, nameB, time.Now().Format("02 January 2006")))
    f.SetCellStyle(sheet, "A2", "J2", subStyle)
    f.SetRowHeight(sheet, 2, 16)

    headers := []string{"No", "NIP", "Nama Pegawai", "Divisi", "Jabatan", fmt.Sprintf("Skor (%s)", nameA), fmt.Sprintf("Grade (%s)", nameA), fmt.Sprintf("Skor (%s)", nameB), fmt.Sprintf("Grade (%s)", nameB), "Tren"}
    for i, h := range headers {
        cell, _ := excelize.CoordinatesToCellName(i+1, 3)
        f.SetCellValue(sheet, cell, h)
        f.SetCellStyle(sheet, cell, cell, headerStyle)
    }
    f.SetRowHeight(sheet, 3, 32)

    for i, row := range rows {
        r := i + 4
        f.SetCellValue(sheet, fmt.Sprintf("A%d", r), i+1)
        f.SetCellValue(sheet, fmt.Sprintf("B%d", r), row.NIP)
        f.SetCellValue(sheet, fmt.Sprintf("C%d", r), row.Name)
        f.SetCellValue(sheet, fmt.Sprintf("D%d", r), row.Division)
        f.SetCellValue(sheet, fmt.Sprintf("E%d", r), row.Position)
        f.SetCellValue(sheet, fmt.Sprintf("F%d", r), row.ScoreA)
        f.SetCellValue(sheet, fmt.Sprintf("G%d", r), row.GradeA)
        f.SetCellValue(sheet, fmt.Sprintf("H%d", r), row.ScoreB)
        f.SetCellValue(sheet, fmt.Sprintf("I%d", r), row.GradeB)
        f.SetCellValue(sheet, fmt.Sprintf("J%d", r), row.Trend)
        f.SetCellStyle(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("A%d", r), centerStyle)
        f.SetCellStyle(sheet, fmt.Sprintf("B%d", r), fmt.Sprintf("I%d", r), baseStyle)
        if ts, ok := trendStyles[row.Trend]; ok {
            f.SetCellStyle(sheet, fmt.Sprintf("J%d", r), fmt.Sprintf("J%d", r), ts)
        }
        f.SetRowHeight(sheet, r, 18)
    }

    f.SetColWidth(sheet, "A", "A", 5)
    f.SetColWidth(sheet, "B", "B", 15)
    f.SetColWidth(sheet, "C", "C", 26)
    f.SetColWidth(sheet, "D", "D", 18)
    f.SetColWidth(sheet, "E", "E", 18)
    f.SetColWidth(sheet, "F", "F", 14)
    f.SetColWidth(sheet, "G", "G", 14)
    f.SetColWidth(sheet, "H", "H", 14)
    f.SetColWidth(sheet, "I", "I", 14)
    f.SetColWidth(sheet, "J", "J", 10)

    filename := fmt.Sprintf("Komparasi_KPI_%s.xlsx", time.Now().Format("20060102"))
    c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
    c.Header("Cache-Control", "no-cache")
    if err := f.Write(c.Writer); err != nil {
        Response(c, http.StatusInternalServerError, "Gagal menulis file Excel komparasi", nil)
    }
}
