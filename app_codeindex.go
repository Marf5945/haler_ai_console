package main

import (
	domain "ui_console/domain/codeindex"
)

func (a *App) RebuildCodeIndex() (interface{}, error) {
	result, err := a.codeIndexService.Rebuild()
	return frontendDTO(result), err
}

func (a *App) SearchCodeSections(query string, limit int) (interface{}, error) {
	matches, err := a.codeIndexService.Search(query, domain.QueryOptions{
		Limit:          limit,
		IncludeRelated: true,
	})
	return frontendDTO(matches), err
}

func (a *App) BuildCodeContext(query string, isHighImpact bool) (interface{}, error) {
	result, err := a.codeIndexService.BuildContext(query, domain.QueryOptions{
		Limit:           5,
		IncludeRelated:  true,
		ContextBefore:   3,
		ContextAfter:    5,
		MaxContextLines: 140,
		MaxContextBytes: 6 * 1024,
	}, isHighImpact)
	return frontendDTO(result), err
}
