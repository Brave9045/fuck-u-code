// Package report 提供代码质量分析报告生成功能
// 创建者：Done-0
// 创建时间：2023-10-01
package report

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/fatih/color"

	"github.com/Done-0/fuck-u-code/pkg/analyzer"
	"github.com/Done-0/fuck-u-code/pkg/i18n"
)

// 颜色风格定义
var (
	titleStyle   = color.New(color.FgHiYellow, color.Bold)
	scoreStyle   = color.New(color.FgHiCyan, color.Bold)
	goodStyle    = color.New(color.FgHiGreen)
	warningStyle = color.New(color.FgHiYellow)
	dangerStyle  = color.New(color.FgHiRed)
	headerStyle  = color.New(color.FgMagenta, color.Bold)
	sectionStyle = color.New(color.FgHiMagenta, color.Bold)
	infoStyle    = color.New(color.FgBlue)
	successStyle = color.New(color.FgGreen, color.Bold)
	detailStyle  = color.New(color.FgCyan)
	metricStyle  = color.New(color.FgCyan)
	fileStyle    = color.New(color.FgMagenta)
	levelStyle   = color.New(color.FgCyan)
	numberStyle  = color.New(color.FgHiWhite)
)

// QualityLevels 定义代码质量等级（每10分一个段位）
var QualityLevels = []struct {
	MinScore    float64
	NameKey     string
	Description string
	Emoji       string
}{
	{0, "level.clean", "level.clean.description", "🌱"},
	{10, "level.mild", "level.mild.description", "🌸"},
	{20, "level.moderate", "level.moderate.description", "😐"},
	{30, "level.bad", "level.bad.description", "😷"},
	{40, "level.terrible", "level.terrible.description", "💩"},
	{50, "level.disaster", "level.disaster.description", "🤕"},
	{60, "level.disaster.severe", "level.disaster.severe.description", "☣️"},
	{70, "level.disaster.very_bad", "level.disaster.very_bad.description", "🧟"},
	{80, "level.disaster.extreme", "level.disaster.extreme.description", "☢️"},
	{90, "level.disaster.worst", "level.disaster.worst.description", "🪦"},
	{100, "level.disaster.ultimate", "level.disaster.ultimate.description", "👑💩"},
}

// Report 表示代码分析报告对象
type Report struct {
	result     *analyzer.AnalysisResult
	translator i18n.Translator
}

// NewReport 创建一个新的报告实例
func NewReport(result *analyzer.AnalysisResult) *Report {
	return &Report{
		result:     result,
		translator: i18n.NewTranslator(i18n.ZhCN), // 默认使用中文
	}
}

// SetTranslator 设置翻译器
func (r *Report) SetTranslator(translator i18n.Translator) {
	r.translator = translator
}

// ReportOptions 定义报告生成的选项
type ReportOptions struct {
	Verbose     bool // 是否显示详细报告
	TopFiles    int  // 显示最差文件的数量
	MaxIssues   int  // 每个文件显示的问题数量
	SummaryOnly bool // 是否只显示摘要
}

// DefaultReportOptions 默认报告选项
var DefaultReportOptions = &ReportOptions{
	Verbose:     false,
	TopFiles:    3,
	MaxIssues:   3,
	SummaryOnly: false,
}

// GenerateConsoleReport 生成控制台报告
func (r *Report) GenerateConsoleReport(options *ReportOptions) {
	if options == nil {
		options = DefaultReportOptions
	}

	score := r.result.CodeQualityScore
	level := r.getQualityLevel(score)

	// 打印标题和总体评分
	printDivider()
	titleStyle.Printf("\n  %s %s %s\n", level.Emoji, r.translator.Translate("report.title"), level.Emoji)
	printDivider()

	fmt.Printf("\n")
	// 精确到小数点后2位
	scoreStyle.Printf("  %s", r.translator.Translate("report.overall_score", math.Round(score*10000)/100))
	fmt.Printf(" - ")
	r.printScoreComment(score)
	fmt.Printf("\n")

	// 打印质量等级
	detailStyle.Printf("  %s", r.translator.Translate("report.level", r.translator.Translate(level.NameKey)))
	detailStyle.Printf(" - %s\n\n", r.translator.Translate(level.Description))

	if !options.SummaryOnly {
		r.printMetricItems() // 打印各项评分指标

		// 详细模式下显示所有文件，否则只显示最差的几个
		if options.Verbose {
			r.printAllFiles(options)
		} else {
			r.printTopIssues(options) // 打印最多问题的代码
		}
	}

	r.printSummary(level) // 打印总结建议

	if options.Verbose {
		r.printVerboseInfo()
	}

	printDivider()
	fmt.Println()
}

// printDivider 打印分隔线
func printDivider() {
	fmt.Printf("\n%s\n", strings.Repeat("─", 80))
}

// printMetricItems 打印各项评分指标及简评
func (r *Report) printMetricItems() {
	sectionStyle.Printf("\n◆ %s\n\n", r.translator.Translate("report.metrics_details"))

	metrics := r.getSortedMetrics()

	// 计算对齐所需的宽度
	maxNameLen := 0
	for _, m := range metrics {
		if len(m.Name) > maxNameLen {
			maxNameLen = len(m.Name)
		}
	}

	// 格式化模板
	nameFormat := fmt.Sprintf("  %%s %%-%ds", maxNameLen+2)
	scoreFormat := "%-8s"

	// 计算总权重和加权分数，用于后续显示
	var totalWeight float64
	var weightedScore float64

	for _, m := range metrics {
		totalWeight += m.Weight
		weightedScore += m.Score * m.Weight
	}

	for _, m := range metrics {
		// 精确到小数点后2位
		scorePercentage := math.Round(m.Score*10000) / 100

		// 确定状态图标和颜色
		var statusEmoji string
		var statusColor *color.Color

		if scorePercentage < 30 {
			statusEmoji = "✓"
			statusColor = goodStyle
		} else if scorePercentage < 70 {
			statusEmoji = "!"
			statusColor = warningStyle
		} else {
			statusEmoji = "✗"
			statusColor = dangerStyle
		}

		// 格式化分数 - 直接显示百分比分数，精确到小数点后2位
		scoreStr := fmt.Sprintf("%.2f%s", scorePercentage, r.translator.Translate("metric.score.suffix"))

		// 打印一行，确保对齐
		statusColor.Printf(nameFormat, statusEmoji, m.Name)
		metricStyle.Printf(scoreFormat, scoreStr)
		detailStyle.Printf("  %s\n", r.getMetricComment(m.Name, scorePercentage))
	}
	fmt.Println()

	// 添加评分计算清单
	if totalWeight > 0 {
		infoStyle.Printf("  %s", r.translator.Translate("report.score_calc"))

		// 计算公式的第一部分 - 显示加权分数
		first := true
		infoStyle.Printf("(")
		for _, m := range metrics {
			if !first {
				infoStyle.Printf(" + ")
			}

			// 精确到小数点后2位
			scorePercentage := math.Round(m.Score*10000) / 100
			infoStyle.Printf("%.2f×%.2f", scorePercentage, m.Weight)

			first = false
		}

		// 计算公式的第二部分 - 显示总权重和最终结果
		// 精确到小数点后2位
		overallScore := math.Round(weightedScore/totalWeight*10000) / 100
		infoStyle.Printf(") ÷ %.2f = %.2f\n\n", totalWeight, overallScore)
	}
}

// getMetricComment 返回指标评论
func (r *Report) getMetricComment(metricName string, score float64) string {
	// 根据指标名称和分数返回对应的评价
	var commentKey string

	// 确定评价级别
	var level string
	if score < 30 {
		level = "good"
	} else if score < 70 {
		level = "medium"
	} else {
		level = "bad"
	}

	// 根据指标类型选择评价
	nameKey := strings.ToLower(metricName)

	// 根据语言和指标类型选择评价
	switch r.translator.GetLanguage() {
	case i18n.EnUS:
		switch {
		case strings.Contains(nameKey, "complexity"):
			commentKey = "metric.complexity." + level
		case strings.Contains(nameKey, "function") || strings.Contains(nameKey, "length"):
			commentKey = "metric.length." + level
		case strings.Contains(nameKey, "comment"):
			commentKey = "metric.comment." + level
		case strings.Contains(nameKey, "error"):
			commentKey = "metric.error." + level
		case strings.Contains(nameKey, "naming"):
			commentKey = "metric.naming." + level
		case strings.Contains(nameKey, "duplication"):
			commentKey = "metric.duplication." + level
		case strings.Contains(nameKey, "structure"):
			commentKey = "metric.structure." + level
		default:
			// 默认评价
			if score < 30 {
				return "Like a spring breeze, code kissed by angels"
			} else if score < 70 {
				return "Not bad, not great, perfectly balanced"
			} else {
				return "Needs serious improvement, like yesterday"
			}
		}
	default: // 中文版本
		switch {
		case strings.Contains(nameKey, "复杂度"):
			commentKey = "metric.complexity." + level
		case strings.Contains(nameKey, "长度"):
			commentKey = "metric.length." + level
		case strings.Contains(nameKey, "注释"):
			commentKey = "metric.comment." + level
		case strings.Contains(nameKey, "错误"):
			commentKey = "metric.error." + level
		case strings.Contains(nameKey, "命名"):
			commentKey = "metric.naming." + level
		case strings.Contains(nameKey, "重复"):
			commentKey = "metric.duplication." + level
		case strings.Contains(nameKey, "结构"):
			commentKey = "metric.structure." + level
		default:
			// 默认评价
			if score < 30 {
				return "如沐春风，代码仿佛被天使亲吻过"
			} else if score < 70 {
				return "不咸不淡，刚刚好，就像人生的平凡日子"
			} else {
				return "惨不忍睹，建议重写，或者假装没看见"
			}
		}
	}

	return r.translator.Translate(commentKey)
}

// printScoreComment 根据得分打印带颜色的总评
func (r *Report) printScoreComment(score float64) {
	comment := r.getScoreComment(score)

	switch {
	case score < 30:
		goodStyle.Printf("%s", comment)
	case score < 70:
		warningStyle.Printf("%s", comment)
	default:
		dangerStyle.Printf("%s", comment)
	}
}

// printTopIssues 打印问题最多的几个代码文件及其问题
func (r *Report) printTopIssues(options *ReportOptions) {
	sectionStyle.Printf("\n◆ %s\n\n", r.translator.Translate("report.worst_files"))

	// 获取所有文件，按问题数量排序
	allFiles := r.getSortedFiles()

	// 如果没有文件，显示提示信息
	if len(allFiles) == 0 {
		successStyle.Println("  🎉 " + r.translator.Translate("report.no_issues"))
		return
	}

	// 计算文件路径最大长度，用于对齐
	maxPathLen := 0
	for _, file := range allFiles {
		pathLen := len(shortenPath(file.FilePath))
		if pathLen > maxPathLen {
			maxPathLen = pathLen
		}
	}

	// 限制最大宽度
	maxPathLen = min(maxPathLen, 60)

	// 确定要显示的文件数量
	maxFiles := min(options.TopFiles, len(allFiles))

	// 处理每个文件
	for i := 0; i < maxFiles; i++ {
		f := allFiles[i]

		// 根据得分选择颜色
		fileScoreColor := getScoreColor(f.FileScore)

		// 打印文件名和得分，精确到小数点后2位
		fmt.Printf("  ")
		numberStyle.Printf("%d. ", i+1)
		fileStyle.Printf("%-*s", maxPathLen+2, shortenPath(f.FilePath))
		fileScoreColor.Printf("(%s)\n", r.translator.Translate("report.file_score", math.Round(f.FileScore*10000)/100))

		// 分类统计问题
		issuesByCategory := r.categorizeIssues(f.Issues)

		// 打印问题分类统计 - 使用更紧凑美观的布局
		if len(issuesByCategory) > 0 {
			// 定义优雅的颜色组合和图标
			categoryInfo := map[string]struct {
				Color *color.Color
				Icon  string
			}{
				"complexity":  {color.New(color.FgMagenta), "🔄 "},
				"comment":     {color.New(color.FgBlue), "📝 "},
				"naming":      {color.New(color.FgCyan), "🏷️  "},
				"structure":   {color.New(color.FgYellow), "🏗️  "},
				"duplication": {color.New(color.FgRed), "📋 "},
				"error":       {color.New(color.FgHiRed), "❌ "},
				"other":       {color.New(color.FgHiYellow), "⚠️  "},
			}

			// 定义问题类别的显示顺序
			categoryOrder := []string{"complexity", "comment", "naming", "structure", "duplication", "error", "other"}

			// 创建一个紧凑的类别统计字符串
			var categories []string
			for _, category := range categoryOrder {
				if count, exists := issuesByCategory[category]; exists {
					// 使用字符串构建器创建每个类别的显示
					var categoryStr strings.Builder

					// 使用颜色写入图标和类别名称
					info := categoryInfo[category]
					categoryStr.WriteString(info.Icon)
					categoryStr.WriteString(r.translator.Translate("issue.category." + category))
					categoryStr.WriteString(": ")

					// 添加到类别列表
					categories = append(categories, fmt.Sprintf("%s%d", categoryStr.String(), count))
				}
			}

			// 计算每行显示的类别数量
			categoriesPerLine := 3
			if len(categories) <= 2 {
				categoriesPerLine = len(categories)
			}

			// 打印类别统计
			indent := "     "
			for i, category := range categories {
				if i > 0 && i%categoriesPerLine == 0 {
					fmt.Printf("\n%s", indent)
				} else if i > 0 {
					fmt.Printf("   ")
				} else {
					fmt.Printf("%s", indent)
				}

				// 解析类别字符串并使用适当的颜色打印
				parts := strings.SplitN(category, ":", 2)
				if len(parts) == 2 {
					// 找出对应的类别以获取颜色
					for catName, info := range categoryInfo {
						catKey := "issue.category." + catName
						catTrans := r.translator.Translate(catKey)

						if strings.Contains(parts[0], catTrans) {
							// 使用颜色打印类别名称和图标
							info.Color.Printf("%s:", parts[0])
							// 使用数字样式打印计数
							numberStyle.Printf("%s", parts[1])
							break
						}
					}
				} else {
					// 回退方案
					fmt.Printf("%s", category)
				}
			}
			fmt.Println()
		}

		// 打印问题列表
		fmt.Println()
		indent := "     "

		if len(f.Issues) == 0 {
			// 如果没有问题，显示"无问题"提示，手动添加✓图标
			successStyle.Printf("%s✓ %s\n", indent, r.translator.Translate("verbose.file_good_quality"))
		} else {
			// 确定显示多少问题
			maxIssues := min(options.MaxIssues, len(f.Issues))

			// 打印问题
			for j := 0; j < maxIssues; j++ {
				// 根据问题类型选择不同图标和颜色
				issueIcon, issueColor := r.getIssueIconAndColor(f.Issues[j])
				fmt.Printf("%s", indent)
				issueColor.Printf("%s%s\n", issueIcon, f.Issues[j])
			}

			// 只在非详细模式下显示"还有更多问题"的提示
			if !options.Verbose && len(f.Issues) > maxIssues {
				warningStyle.Printf("%s🔍 %s\n",
					indent, r.translator.Translate("report.more_issues", len(f.Issues)-maxIssues))
			}
		}

		if i < maxFiles-1 {
			fmt.Println()
		}
	}
}

// categorizeIssues 将问题按类别分类统计
func (r *Report) categorizeIssues(issues []string) map[string]int {
	categories := map[string]int{
		"complexity":  0, // 复杂度问题
		"comment":     0, // 注释问题
		"naming":      0, // 命名问题
		"structure":   0, // 结构问题
		"duplication": 0, // 重复问题
		"error":       0, // 错误处理问题
		"other":       0, // 其他问题
	}

	for _, issue := range issues {
		lowerIssue := strings.ToLower(issue)

		switch {
		case strings.Contains(lowerIssue, "复杂度") || strings.Contains(lowerIssue, "complexity"):
			categories["complexity"]++
		case strings.Contains(lowerIssue, "注释") || strings.Contains(lowerIssue, "comment"):
			categories["comment"]++
		case strings.Contains(lowerIssue, "命名") || strings.Contains(lowerIssue, "name") || strings.Contains(lowerIssue, "naming"):
			categories["naming"]++
		case strings.Contains(lowerIssue, "结构") || strings.Contains(lowerIssue, "嵌套") || strings.Contains(lowerIssue, "structure") || strings.Contains(lowerIssue, "nest"):
			categories["structure"]++
		case strings.Contains(lowerIssue, "重复") || strings.Contains(lowerIssue, "duplication"):
			categories["duplication"]++
		case strings.Contains(lowerIssue, "错误") || strings.Contains(lowerIssue, "error"):
			categories["error"]++
		default:
			categories["other"]++
		}
	}

	// 删除计数为0的类别
	for category, count := range categories {
		if count == 0 {
			delete(categories, category)
		}
	}

	return categories
}

// getIssueIconAndColor 根据问题内容返回合适的图标和颜色
func (r *Report) getIssueIconAndColor(issue string) (string, *color.Color) {
	lowerIssue := strings.ToLower(issue)

	switch {
	case strings.Contains(lowerIssue, "复杂度") || strings.Contains(lowerIssue, "complexity"):
		return "🔄 ", color.New(color.FgMagenta) // 窄图标，只需一个空格
	case strings.Contains(lowerIssue, "注释") || strings.Contains(lowerIssue, "comment"):
		return "📝 ", color.New(color.FgBlue) // 窄图标，只需一个空格
	case strings.Contains(lowerIssue, "命名") || strings.Contains(lowerIssue, "name") || strings.Contains(lowerIssue, "naming"):
		return "🏷️  ", color.New(color.FgCyan) // 宽图标，需要两个空格
	case strings.Contains(lowerIssue, "结构") || strings.Contains(lowerIssue, "嵌套") || strings.Contains(lowerIssue, "structure") || strings.Contains(lowerIssue, "nest"):
		return "🏗️  ", color.New(color.FgYellow) // 宽图标，需要两个空格
	case strings.Contains(lowerIssue, "重复") || strings.Contains(lowerIssue, "duplication"):
		return "📋 ", color.New(color.FgRed) // 窄图标，只需一个空格
	case strings.Contains(lowerIssue, "错误") || strings.Contains(lowerIssue, "error"):
		return "❌ ", color.New(color.FgHiRed) // 窄图标，只需一个空格
	default:
		return "⚠️  ", color.New(color.FgHiYellow) // 宽图标，需要两个空格
	}
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// shortenPath 缩短文件路径，只显示最后几个部分
func shortenPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 4 {
		return path
	}

	return "./" + strings.Join(parts[len(parts)-3:], "/")
}

// printSummary 打印最终诊断结论和建议
func (r *Report) printSummary(level struct {
	MinScore    float64
	NameKey     string
	Description string
	Emoji       string
}) {
	sectionStyle.Printf("\n◆ %s\n\n", r.translator.Translate("report.conclusion"))

	// 使用levelStyle打印等级名称和表情符号
	fmt.Printf("  %s ", level.Emoji)
	levelStyle.Printf("%s", r.translator.Translate(level.NameKey))
	detailStyle.Printf(" - %s\n\n", r.translator.Translate(level.Description))

	// 根据不同等级提供相应的建议
	switch {
	case level.MinScore < 30:
		successStyle.Println("  " + r.translator.Translate("advice.good"))
	case level.MinScore < 60:
		warningStyle.Println("  " + r.translator.Translate("advice.moderate"))
	default:
		dangerStyle.Println("  " + r.translator.Translate("advice.bad"))
	}
	fmt.Println()
}

// getScoreComment 根据得分生成总评
func (r *Report) getScoreComment(score float64) string {
	score = score * 100

	// 确定分数区间，每10分一个区间
	scoreRange := int(score) / 10 * 10
	if scoreRange > 90 {
		scoreRange = 90
	}

	commentKey := fmt.Sprintf("score.comment.%d", scoreRange)
	return r.translator.Translate(commentKey)
}

// getSortedMetrics 按照分数升序排列各项指标
func (r *Report) getSortedMetrics() []analyzer.MetricResult {
	var metrics []analyzer.MetricResult
	for _, m := range r.result.Metrics {
		metrics = append(metrics, m)
	}
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Score < metrics[j].Score
	})
	return metrics
}

// getSortedFiles 获取按分数排序的问题文件列表（降序）
func (r *Report) getSortedFiles() []analyzer.FileAnalysisResult {
	worstFiles := append([]analyzer.FileAnalysisResult{}, r.result.FilesAnalyzed...)
	sort.Slice(worstFiles, func(i, j int) bool {
		return worstFiles[i].FileScore > worstFiles[j].FileScore
	})
	return worstFiles
}

// getQualityLevel 根据总分获取对应的质量等级
func (r *Report) getQualityLevel(score float64) struct {
	MinScore    float64
	NameKey     string
	Description string
	Emoji       string
} {
	level := QualityLevels[0]
	for i := len(QualityLevels) - 1; i >= 0; i-- {
		if score >= QualityLevels[i].MinScore {
			level = QualityLevels[i]
			break
		}
	}
	return level
}

// printVerboseInfo 打印详细的代码分析信息
func (r *Report) printVerboseInfo() {
	sectionStyle.Printf("\n◆ %s\n\n", r.translator.Translate("verbose.basic_statistics"))

	// 打印基本统计数据
	headerStyle.Println("  📊 " + r.translator.Translate("verbose.basic_statistics"))
	detailStyle.Printf("    %-15s %d\n", r.translator.Translate("verbose.total_files"), r.result.TotalFiles)
	detailStyle.Printf("    %-15s %d\n", r.translator.Translate("verbose.total_lines"), r.result.TotalLines)
	detailStyle.Printf("    %-15s %d\n", r.translator.Translate("verbose.total_issues"), r.getTotalIssues())

	// 打印各指标详细信息
	headerStyle.Println("\n  🔍 " + r.translator.Translate("verbose.metric_details"))

	metrics := r.getSortedMetrics()
	maxNameLen := 0
	for _, metric := range metrics {
		if len(metric.Name) > maxNameLen {
			maxNameLen = len(metric.Name)
		}
	}

	nameFormat := fmt.Sprintf("\n    【%%-%ds】", maxNameLen)

	for _, metric := range metrics {
		scoreColor := getScoreColor(metric.Score)
		metricStyle.Printf(nameFormat, metric.Name)
		infoStyle.Printf("(%s %.2f)\n", r.translator.Translate("verbose.weight"), metric.Weight)
		detailStyle.Printf("      %s %s\n", r.translator.Translate("verbose.description"), metric.Description)
		fmt.Printf("      %s ", r.translator.Translate("verbose.score"))
		// 精确到小数点后2位
		scoreColor.Printf("%.2f/100\n", math.Round(metric.Score*10000)/100)
	}
}

// getTotalIssues 获取所有文件的问题总数
func (r *Report) getTotalIssues() int {
	total := 0
	for _, file := range r.result.FilesAnalyzed {
		total += len(file.Issues)
	}
	return total
}

// getScoreColor 根据得分返回对应的颜色
func getScoreColor(score float64) *color.Color {
	switch {
	case score > 0.7:
		return dangerStyle
	case score > 0.3:
		return warningStyle
	default:
		return goodStyle
	}
}

// printAllFiles 打印所有文件及其问题
func (r *Report) printAllFiles(options *ReportOptions) {
	sectionStyle.Printf("\n◆ %s\n\n", r.translator.Translate("verbose.all_files"))

	files := r.getSortedFiles()
	if len(files) == 0 {
		successStyle.Println("  " + r.translator.Translate("verbose.no_files_found"))
		return
	}

	// 计算文件路径最大长度，用于对齐
	maxPathLen := 0
	for _, file := range files {
		pathLen := len(shortenPath(file.FilePath))
		if pathLen > maxPathLen {
			maxPathLen = pathLen
		}
	}

	// 限制最大宽度
	maxPathLen = min(maxPathLen, 60)

	// 根据options.TopFiles决定显示多少文件
	maxFilesToShow := len(files)
	if !options.Verbose && options.TopFiles > 0 && options.TopFiles < maxFilesToShow {
		maxFilesToShow = options.TopFiles
	}

	for i, f := range files[:maxFilesToShow] {
		// 根据得分选择颜色
		fileScoreColor := getScoreColor(f.FileScore)

		// 打印文件名和得分
		fmt.Printf("  ")
		numberStyle.Printf("%d. ", i+1)
		fileStyle.Printf("%-*s", maxPathLen+2, shortenPath(f.FilePath))
		fileScoreColor.Printf("(%s)\n", r.translator.Translate("report.file_score", math.Round(f.FileScore*10000)/100))

		// 分类统计问题
		issuesByCategory := r.categorizeIssues(f.Issues)

		// 打印问题分类统计 - 使用更紧凑美观的布局
		if len(issuesByCategory) > 0 {
			// 定义优雅的颜色组合和图标
			categoryInfo := map[string]struct {
				Color *color.Color
				Icon  string
			}{
				"complexity":  {color.New(color.FgMagenta), "🔄 "},
				"comment":     {color.New(color.FgBlue), "📝 "},
				"naming":      {color.New(color.FgCyan), "🏷️  "},
				"structure":   {color.New(color.FgYellow), "🏗️  "},
				"duplication": {color.New(color.FgRed), "📋 "},
				"error":       {color.New(color.FgHiRed), "❌ "},
				"other":       {color.New(color.FgHiYellow), "⚠️  "},
			}

			// 定义问题类别的显示顺序
			categoryOrder := []string{"complexity", "comment", "naming", "structure", "duplication", "error", "other"}

			// 创建一个紧凑的类别统计字符串
			var categories []string
			for _, category := range categoryOrder {
				if count, exists := issuesByCategory[category]; exists {
					// 使用字符串构建器创建每个类别的显示
					var categoryStr strings.Builder

					// 使用颜色写入图标和类别名称
					info := categoryInfo[category]
					categoryStr.WriteString(info.Icon)
					categoryStr.WriteString(r.translator.Translate("issue.category." + category))
					categoryStr.WriteString(": ")

					// 添加到类别列表
					categories = append(categories, fmt.Sprintf("%s%d", categoryStr.String(), count))
				}
			}

			// 计算每行显示的类别数量
			categoriesPerLine := 3
			if len(categories) <= 2 {
				categoriesPerLine = len(categories)
			}

			// 打印类别统计
			indent := "     "
			for i, category := range categories {
				if i > 0 && i%categoriesPerLine == 0 {
					fmt.Printf("\n%s", indent)
				} else if i > 0 {
					fmt.Printf("   ")
				} else {
					fmt.Printf("%s", indent)
				}

				// 解析类别字符串并使用适当的颜色打印
				parts := strings.SplitN(category, ":", 2)
				if len(parts) == 2 {
					// 找出对应的类别以获取颜色
					for catName, info := range categoryInfo {
						catKey := "issue.category." + catName
						catTrans := r.translator.Translate(catKey)

						if strings.Contains(parts[0], catTrans) {
							// 使用颜色打印类别名称和图标
							info.Color.Printf("%s:", parts[0])
							// 使用数字样式打印计数
							numberStyle.Printf("%s", parts[1])
							break
						}
					}
				} else {
					// 回退方案
					fmt.Printf("%s", category)
				}
			}
			fmt.Println()
		}

		// 打印问题列表
		fmt.Println()
		indent := "     "
		if len(f.Issues) == 0 {
			// 如果没有问题，显示"无问题"提示，手动添加✓图标
			successStyle.Printf("%s✓ %s\n", indent, r.translator.Translate("verbose.file_good_quality"))
		} else {
			// 确定显示多少问题
			maxIssues := len(f.Issues)
			if !options.Verbose {
				maxIssues = min(options.MaxIssues, maxIssues)
			}

			for j := 0; j < maxIssues; j++ {
				issueIcon, issueColor := r.getIssueIconAndColor(f.Issues[j])
				fmt.Printf("%s", indent)
				issueColor.Printf("%s%s\n", issueIcon, f.Issues[j])
			}

			// 只在非详细模式下显示"还有更多问题"的提示
			if !options.Verbose && len(f.Issues) > maxIssues {
				warningStyle.Printf("%s🔍 %s\n",
					indent, r.translator.Translate("report.more_issues", len(f.Issues)-maxIssues))
			}
		}

		if i < maxFilesToShow-1 {
			fmt.Println()
		}
	}
}
