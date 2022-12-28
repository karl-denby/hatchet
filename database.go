// Copyright 2022-present Kuei-chun Chen. All rights reserved.

package hatchet

import (
	"database/sql"
	"fmt"
	"strings"
)

func getSlowOps(tableName string, orderBy string, order string, collscan bool) ([]OpStat, error) {
	ops := []OpStat{}
	db, err := sql.Open("sqlite3", SQLITE_FILE)
	if err != nil {
		return ops, err
	}
	defer db.Close()
	query := getSlowOpsQuery(tableName, orderBy, order, collscan)
	rows, err := db.Query(query)
	if err != nil {
		return ops, err
	}
	defer rows.Close()
	for rows.Next() {
		var op OpStat
		if err = rows.Scan(&op.Op, &op.Count, &op.AvgMilli, &op.MaxMilli, &op.TotalMilli,
			&op.Namespace, &op.Index, &op.Reslen, &op.QueryPattern); err != nil {
			return ops, err
		}
		ops = append(ops, op)
	}
	return ops, err
}

func getSlowOpsQuery(tableName string, orderBy string, order string, collscan bool) string {
	query := fmt.Sprintf(`SELECT op, COUNT(*) "count", ROUND(AVG(milli),1) avg_ms, MAX(milli) max_ms, SUM(milli) total_ms,
			ns, _index "index", SUM(reslen) "reslen", filter "query pattern"
			FROM %v WHERE op != "" GROUP BY op, ns, filter ORDER BY %v %v`, tableName, orderBy, order)
	if collscan {
		query = fmt.Sprintf(`SELECT op, COUNT(*) "count", ROUND(AVG(milli),1) avg_ms, MAX(milli) max_ms, SUM(milli) total_ms,
				ns, _index "index", SUM(reslen) "reslen", filter "query pattern"
				FROM %v WHERE op != "" AND _index = "COLLSCAN" GROUP BY op, ns, filter ORDER BY %v %v`, tableName, orderBy, order)
	}
	return query
}

func getLogs(tableName string, opts ...string) ([]LegacyLog, error) {
	docs := []LegacyLog{}
	query := fmt.Sprintf(`SELECT date, severity, component, context, message FROM %v`, tableName)
	if len(opts) > 0 {
		query += " WHERE"
		cnt := 0
		for _, opt := range opts {
			toks := strings.Split(opt, "=")
			if len(toks) < 2 || toks[1] == "" {
				continue
			}
			if cnt > 0 {
				query += " AND"
			}
			if toks[0] == "duration" {
				dates := strings.Split(toks[1], ",")
				query += fmt.Sprintf(" date BETWEEN '%v' and '%v'", dates[0], dates[1])
			} else {
				query += fmt.Sprintf(" %v = '%v'", toks[0], toks[1])
			}
			cnt++
		}
	}
	db, err := sql.Open("sqlite3", SQLITE_FILE)
	if err != nil {
		return docs, err
	}
	defer db.Close()
	rows, err := db.Query(query)
	if err != nil {
		return docs, err
	}
	defer rows.Close()
	for rows.Next() {
		var doc LegacyLog
		if err = rows.Scan(&doc.Timestamp, &doc.Severity, &doc.Component, &doc.Context, &doc.Message); err != nil {
			return docs, err
		}
		docs = append(docs, doc)
	}
	return docs, err
}

func getSlowestLogs(tableName string, topN int) ([]string, error) {
	logstrs := []string{}
	query := fmt.Sprintf(`SELECT date, severity, component, context, message
			FROM %v WHERE op != "" ORDER BY milli DESC LIMIT %v`, tableName, topN)
	db, err := sql.Open("sqlite3", SQLITE_FILE)
	if err != nil {
		return logstrs, err
	}
	defer db.Close()
	rows, err := db.Query(query)
	if err != nil {
		return logstrs, err
	}
	defer rows.Close()
	for rows.Next() {
		var doc Logv2Info
		var date string
		if err = rows.Scan(&date, &doc.Severity, &doc.Component, &doc.Context, &doc.Message); err != nil {
			return logstrs, err
		}
		logstr := fmt.Sprintf("%v %-2s %-8s [%v] %v", date, doc.Severity, doc.Component, doc.Context, doc.Message)
		logstrs = append(logstrs, logstr)
	}
	return logstrs, err
}
