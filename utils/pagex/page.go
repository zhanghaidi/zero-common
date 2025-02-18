package pagex

const (
	DefaultPageSize = 10
	MaxPageSize     = 10000
)

// InitPage 初始化分页，并计算最后一页
func InitPage(reqPage, reqPageSize int, total int64) (page, pageSize, lastPage int) {
	// 默认分页设置
	pageSize = DefaultPageSize
	page = 1

	// 设置页面大小
	if reqPageSize > 0 && reqPageSize <= MaxPageSize {
		pageSize = reqPageSize
	}

	// 设置当前页
	if reqPage > 1 {
		page = reqPage
	}

	// 计算总页数
	lastPage = (int(total) + pageSize - 1) / pageSize

	// 如果请求的页码大于最后一页，则将页码设置为最后一页
	if page > lastPage && lastPage > 0 {
		page = lastPage
	}

	return
}

// GetPaperIndexPage 计算试卷的页码和索引
func GetPaperIndexPage(progress, pageSize int) (progressPage, index int) {
	if progress == 0 || pageSize <= 0 {
		return 0, 0
	}

	progressPage = (progress / pageSize) + 1

	index = progress % pageSize

	return progressPage, index
}
