package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	config "github.com/mdhasib01/go-rest-starter/config"
	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"

	"github.com/xuri/excelize/v2"
)

func DeativateProfiles(session *model.Session) {
	session.BuyerProfileId = 0
	session.SellerProfileId = 0
}

func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func GetPhotoUrl(photo string) string {
	return config.Param.ServerBaseURL + "/static/" + photo

}
func getPaginationParams(query url.Values) (page int, size int) {
	page = 1
	size = 10
	if p, err := strconv.Atoi(query.Get("page")); err == nil {
		page = p
		if page < 1 {
			page = 1
		}
	}
	if s, err := strconv.Atoi(query.Get("size")); err == nil {
		size = s
		if size < 1 {
			size = 10
		}
	}
	return page, size
}

func MakeRequest(method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	// Create a new HTTP request
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Set headers
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	// Create an HTTP client
	client := &http.Client{}

	// Make the request
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

var chars = []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
	"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}

func GetLoggedProfileId(session model.Session) int {
	var profileId int
	if session.BuyerProfileId != 0 {
		profileId = session.BuyerProfileId
	}
	if session.SellerProfileId != 0 {
		profileId = session.SellerProfileId
	}
	return profileId
}

func HashDocument(doc []byte) [32]byte {
	return sha256.Sum256(doc)
}

func removeExcelFileFormulas(f *excelize.File) (*excelize.File, error) {
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return nil, model.NewError(itn.ErrorUnknown, 500)
		}
		for i, row := range rows {
			for j := range row {
				form, err := f.GetCellFormula(sheet, fmt.Sprintf("%s%d", chars[j], i+1))
				if err != nil {
					logger.GetLogger().LogErrors(err, nil)
					return nil, model.NewError(itn.ErrorUnknown, 500)
				}
				if form != "" {
					err = f.SetCellValue(sheet, fmt.Sprintf("%s%d", chars[j], i+1), "")
					if err != nil {
						logger.GetLogger().LogErrors(err, nil)
						return nil, model.NewError(itn.ErrorUnknown, 500)
					}
				}
			}
		}
	}

	return f, nil
}

func openExcelFiles(path string) (*excelize.File, *excelize.File, error) {
	file, err := excelize.OpenFile(path)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return nil, nil, model.NewError(itn.ErrorUnknown, 500)
	}

	emptyFile, err := excelize.OpenFile(path)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return nil, nil, model.NewError(itn.ErrorUnknown, 500)
	}

	return file, emptyFile, nil
}

func PrepareFinancialLeverage(file, emptyFile *excelize.File, h model.ExcelSheetRequest, s int) error {
	i := 1

	sheetName := "Levier financier"

	for _, y := range h.PreviousYears {
		start := chars[i]
		emptyFile.SetCellValue(sheetName, start+"3", y.Year)
		emptyFile.SetCellStyle(sheetName, start+"3", start+"3", s)

		result, err := file.CalcCellValue(sheetName, start+"4")
		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, start+"4", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}
		res, _ := strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)

		emptyFile.SetCellValue(sheetName, start+"4", result)
		emptyFile.SetCellStyle(sheetName, start+"4", start+"4", s)

		emptyFile.SetCellValue(sheetName, start+"7", y.Year)
		emptyFile.SetCellStyle(sheetName, start+"7", start+"7", s)

		result, err = file.CalcCellValue(sheetName, start+"8")
		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, start+"8", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}
		res, _ = strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)
		file.SetCellValue(sheetName, start+"8", result)
		emptyFile.SetCellValue(sheetName, start+"8", result)
		emptyFile.SetCellStyle(sheetName, start+"8", start+"8", s)

		emptyFile.SetCellValue(sheetName, start+"11", y.Year)
		emptyFile.SetCellStyle(sheetName, start+"11", start+"11", s)

		result, err = file.CalcCellValue(sheetName, start+"12")
		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, start+"12", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}
		res, _ = strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)
		file.SetCellValue(sheetName, start+"12", result)
		emptyFile.SetCellValue(sheetName, start+"12", result)
		emptyFile.SetCellStyle(sheetName, start+"12", start+"12", s)

		i += 2
	}
	return nil
}

func PrepareEfficiencyRatios(file, emptyFile *excelize.File, h model.ExcelSheetRequest, s int) error {
	i := 1
	sheetName := "Ratios d'efficience"
	for _, y := range h.PreviousYears {
		start := chars[i]
		emptyFile.SetCellValue(sheetName, start+"3", y.Year)

		result, err := file.CalcCellValue(sheetName, start+"4")
		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, start+"4", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}
		res, _ := strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)
		file.SetCellValue(sheetName, start+"4", result)
		emptyFile.SetCellValue(sheetName, start+"4", result)
		emptyFile.SetCellStyle(sheetName, start+"4", start+"4", s)

		emptyFile.SetCellValue(sheetName, start+"7", y.Year)

		result, err = file.CalcCellValue(sheetName, start+"8")
		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, start+"8", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}

		res, _ = strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)
		file.SetCellValue(sheetName, start+"8", result)
		emptyFile.SetCellValue(sheetName, start+"8", result)
		emptyFile.SetCellStyle(sheetName, start+"8", start+"8", s)

		emptyFile.SetCellValue(sheetName, start+"12", y.Year)

		result, err = file.CalcCellValue(sheetName, start+"13")
		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, start+"13", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}
		res, _ = strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)
		file.SetCellValue(sheetName, start+"13", result)
		emptyFile.SetCellValue(sheetName, start+"13", result)
		emptyFile.SetCellStyle(sheetName, start+"13", start+"13", s)

		emptyFile.SetCellValue(sheetName, start+"16", y.Year)

		result, err = file.CalcCellValue(sheetName, start+"17")
		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, start+"17", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}
		res, _ = strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)
		file.SetCellValue(sheetName, start+"17", result)
		emptyFile.SetCellValue(sheetName, start+"17", result)
		emptyFile.SetCellStyle(sheetName, start+"17", start+"17", s)

		emptyFile.SetCellValue(sheetName, start+"21", y.Year)

		result, err = file.CalcCellValue(sheetName, start+"22")
		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, start+"22", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}

		res, _ = strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)
		file.SetCellValue(sheetName, start+"22", result)
		emptyFile.SetCellValue(sheetName, start+"22", result)
		emptyFile.SetCellStyle(sheetName, start+"22", start+"22", s)

		emptyFile.SetCellValue(sheetName, start+"25", y.Year)

		result, err = file.CalcCellValue(sheetName, start+"26")
		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, start+"26", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}
		res, _ = strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)
		file.SetCellValue(sheetName, start+"26", result)
		emptyFile.SetCellValue(sheetName, start+"26", result)
		emptyFile.SetCellStyle(sheetName, start+"26", start+"26", s)

		emptyFile.SetCellValue(sheetName, start+"31", y.Year)

		result, err = file.CalcCellValue(sheetName, start+"32")
		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, start+"32", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}
		res, _ = strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)
		file.SetCellValue(sheetName, start+"32", result)
		emptyFile.SetCellValue(sheetName, start+"32", result)
		emptyFile.SetCellStyle(sheetName, start+"32", start+"32", s)

		emptyFile.SetCellValue(sheetName, start+"35", y.Year)

		result, err = file.CalcCellValue(sheetName, start+"36")
		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, start+"36", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}
		res, _ = strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)
		file.SetCellValue(sheetName, start+"36", result)
		emptyFile.SetCellValue(sheetName, start+"36", result)
		emptyFile.SetCellStyle(sheetName, start+"36", start+"36", s)

		i += 2
	}
	return nil
}
func PrepareProfitabilityRatiosSheet(file, emptyFile *excelize.File, h model.ExcelSheetRequest, ratiosStyle int) error {

	i := 1
	sheetName := "Ratios de rentabilité"

	for _, y := range h.PreviousYears {
		start := chars[i]

		emptyFile.SetCellValue(sheetName, start+"3", y.Year)
		emptyFile.SetCellStyle(sheetName, start+"3", start+"3", ratiosStyle)

		result, err := file.CalcCellValue(sheetName, start+"4")
		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, start+"4", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}
		res, _ := strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)
		file.SetCellValue(sheetName, start+"4", result)
		emptyFile.SetCellValue(sheetName, start+"4", result)
		emptyFile.SetCellStyle(sheetName, start+"4", start+"4", ratiosStyle)

		emptyFile.SetCellValue(sheetName, start+"7", y.Year)
		emptyFile.SetCellStyle(sheetName, start+"7", start+"7", ratiosStyle)

		result, err = file.CalcCellValue(sheetName, start+"8")
		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, start+"8", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}
		res, _ = strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)
		file.SetCellValue(sheetName, start+"8", result)
		emptyFile.SetCellValue(sheetName, start+"8", result)
		emptyFile.SetCellStyle(sheetName, start+"8", start+"8", ratiosStyle)

		emptyFile.SetCellValue(sheetName, start+"12", y.Year)
		emptyFile.SetCellStyle(sheetName, start+"12", start+"12", ratiosStyle)

		result, err = file.CalcCellValue(sheetName, start+"13")
		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, start+"13", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}
		res, _ = strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)
		file.SetCellValue(sheetName, start+"13", result)
		emptyFile.SetCellValue(sheetName, start+"13", result)
		emptyFile.SetCellStyle(sheetName, start+"13", start+"13", ratiosStyle)

		emptyFile.SetCellValue(sheetName, start+"16", y.Year)
		emptyFile.SetCellStyle(sheetName, start+"16", start+"16", ratiosStyle)

		result, err = file.CalcCellValue(sheetName, start+"17")
		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, start+"17", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}
		res, _ = strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)
		file.SetCellValue(sheetName, start+"17", result)
		emptyFile.SetCellValue(sheetName, start+"17", result)
		emptyFile.SetCellStyle(sheetName, start+"17", start+"17", ratiosStyle)

		i += 2
	}
	return nil

}

func PrepareLiquidityRatiosSheet(file, emptyFile *excelize.File, h model.ExcelSheetRequest, ratiosStyle int) error {
	i := 1

	sheetName := "Ratios de liquidité"

	for _, y := range h.PreviousYears {
		startChar := chars[i]

		emptyFile.SetCellValue(sheetName, startChar+"3", y.Year)
		emptyFile.SetCellStyle(sheetName, startChar+"3", startChar+"3", ratiosStyle)

		result, err := file.CalcCellValue(sheetName, startChar+"4")

		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, startChar+"4", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}
		res, _ := strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)
		file.SetCellValue(sheetName, startChar+"4", result)
		emptyFile.SetCellValue(sheetName, startChar+"4", result)
		emptyFile.SetCellStyle(sheetName, startChar+"4", startChar+"4", ratiosStyle)

		emptyFile.SetCellValue(sheetName, startChar+"10", y.Year)
		emptyFile.SetCellStyle(sheetName, startChar+"10", startChar+"10", ratiosStyle)

		result, err = file.CalcCellValue(sheetName, startChar+"11")
		if err != nil {
			if strings.Contains(err.Error(), "#DIV/0!") {
				emptyFile.SetCellValue(sheetName, startChar+"11", "")
			} else {
				logger.GetLogger().LogErrors(err, nil)
				return model.NewError(itn.ErrorUnknown, 500)
			}
		}

		res, _ = strconv.ParseFloat(result, 64)
		result = fmt.Sprintf("%.2f", res)
		file.SetCellValue(sheetName, startChar+"11", result)
		emptyFile.SetCellValue(sheetName, startChar+"11", result)
		emptyFile.SetCellStyle(sheetName, startChar+"11", startChar+"11", ratiosStyle)

		i += 2

	}
	return nil
}
