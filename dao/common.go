package dao

import (
	"fmt"
	"strings"

	model "github.com/mdhasib01/go-rest-starter/model"
)

func buildFilteredQuery(query string, fr model.FilteredRequest, args []interface{}, alias string) (string, []interface{}, error) {
	if fr.OrderParams != "" {
		parts := strings.Split(fr.OrderParams, ":")
		if len(parts) == 2 {
			field := parts[0]
			dir := strings.ToUpper(parts[1])
			if dir != "ASC" && dir != "DESC" {
				dir = "ASC"
			}

			// Simple validation: check if field is in AllowedOrderParams
			allowed := false
			for _, f := range fr.AllowedOrderParams {
				if f == field {
					allowed = true
					break
				}
			}
			if allowed {
				if alias != "" {
					query += fmt.Sprintf(" ORDER BY %s.%s %s", alias, field, dir)
				} else {
					query += fmt.Sprintf(" ORDER BY %s %s", field, dir)
				}
			}
		}
	}

	if fr.Size > 0 {
		query += fmt.Sprintf(" LIMIT %d", fr.Size)
		if fr.Page > 0 {
			offset := (fr.Page - 1) * fr.Size
			query += fmt.Sprintf(" OFFSET %d", offset)
		}
	}

	return query, args, nil
}

func GetTableCount(query string, idField string, fr model.FilteredRequest, args []interface{}, alias string) (int, error) {
	upper := strings.ToUpper(query)
	
    // Strip ORDER BY
    idx := strings.LastIndex(upper, " ORDER BY ")
	if idx != -1 {
		query = query[:idx]
		upper = strings.ToUpper(query)
	}
    
    // Strip LIMIT
	idx = strings.LastIndex(upper, " LIMIT ")
	if idx != -1 {
		query = query[:idx]
		upper = strings.ToUpper(query)
	}
    
    // Strip OFFSET
	idx = strings.LastIndex(upper, " OFFSET ")
	if idx != -1 {
		query = query[:idx]
	}

	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS t"

	var count int
	err := DB.QueryRow(countQuery, args...).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
