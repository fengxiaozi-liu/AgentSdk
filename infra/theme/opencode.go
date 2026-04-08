package theme

import "github.com/charmbracelet/lipgloss"

type OpenCodeTheme struct {
	BaseTheme
}

func NewOpenCodeTheme() *OpenCodeTheme {
	darkBackground := "#212121"
	darkCurrentLine := "#252525"
	darkSelection := "#303030"
	darkForeground := "#e0e0e0"
	darkComment := "#6a6a6a"
	darkPrimary := "#fab283"
	darkSecondary := "#5c9cf5"
	darkAccent := "#9d7cd8"
	darkRed := "#e06c75"
	darkOrange := "#f5a742"
	darkGreen := "#7fd88f"
	darkCyan := "#56b6c2"
	darkYellow := "#e5c07b"
	darkBorder := "#4b4c5c"
	lightBackground := "#f8f8f8"
	lightCurrentLine := "#f0f0f0"
	lightSelection := "#e5e5e6"
	lightForeground := "#2a2a2a"
	lightComment := "#8a8a8a"
	lightPrimary := "#3b7dd8"
	lightSecondary := "#7b5bb6"
	lightAccent := "#d68c27"
	lightRed := "#d1383d"
	lightOrange := "#d68c27"
	lightGreen := "#3d9a57"
	lightCyan := "#318795"
	lightYellow := "#b0851f"
	lightBorder := "#d3d3d3"

	theme := &OpenCodeTheme{}
	theme.PrimaryColor = lipgloss.AdaptiveColor{Dark: darkPrimary, Light: lightPrimary}
	theme.SecondaryColor = lipgloss.AdaptiveColor{Dark: darkSecondary, Light: lightSecondary}
	theme.AccentColor = lipgloss.AdaptiveColor{Dark: darkAccent, Light: lightAccent}
	theme.ErrorColor = lipgloss.AdaptiveColor{Dark: darkRed, Light: lightRed}
	theme.WarningColor = lipgloss.AdaptiveColor{Dark: darkOrange, Light: lightOrange}
	theme.SuccessColor = lipgloss.AdaptiveColor{Dark: darkGreen, Light: lightGreen}
	theme.InfoColor = lipgloss.AdaptiveColor{Dark: darkCyan, Light: lightCyan}
	theme.TextColor = lipgloss.AdaptiveColor{Dark: darkForeground, Light: lightForeground}
	theme.TextMutedColor = lipgloss.AdaptiveColor{Dark: darkComment, Light: lightComment}
	theme.TextEmphasizedColor = lipgloss.AdaptiveColor{Dark: darkYellow, Light: lightYellow}
	theme.BackgroundColor = lipgloss.AdaptiveColor{Dark: darkBackground, Light: lightBackground}
	theme.BackgroundSecondaryColor = lipgloss.AdaptiveColor{Dark: darkCurrentLine, Light: lightCurrentLine}
	theme.BackgroundDarkerColor = lipgloss.AdaptiveColor{Dark: "#121212", Light: "#ffffff"}
	theme.BorderNormalColor = lipgloss.AdaptiveColor{Dark: darkBorder, Light: lightBorder}
	theme.BorderFocusedColor = lipgloss.AdaptiveColor{Dark: darkPrimary, Light: lightPrimary}
	theme.BorderDimColor = lipgloss.AdaptiveColor{Dark: darkSelection, Light: lightSelection}
	theme.DiffAddedColor = lipgloss.AdaptiveColor{Dark: "#478247", Light: "#2E7D32"}
	theme.DiffRemovedColor = lipgloss.AdaptiveColor{Dark: "#7C4444", Light: "#C62828"}
	theme.DiffContextColor = lipgloss.AdaptiveColor{Dark: "#a0a0a0", Light: "#757575"}
	theme.DiffHunkHeaderColor = lipgloss.AdaptiveColor{Dark: "#a0a0a0", Light: "#757575"}
	theme.DiffHighlightAddedColor = lipgloss.AdaptiveColor{Dark: "#DAFADA", Light: "#A5D6A7"}
	theme.DiffHighlightRemovedColor = lipgloss.AdaptiveColor{Dark: "#FADADD", Light: "#EF9A9A"}
	theme.DiffAddedBgColor = lipgloss.AdaptiveColor{Dark: "#303A30", Light: "#E8F5E9"}
	theme.DiffRemovedBgColor = lipgloss.AdaptiveColor{Dark: "#3A3030", Light: "#FFEBEE"}
	theme.DiffContextBgColor = lipgloss.AdaptiveColor{Dark: darkBackground, Light: lightBackground}
	theme.DiffLineNumberColor = lipgloss.AdaptiveColor{Dark: "#888888", Light: "#9E9E9E"}
	theme.DiffAddedLineNumberBgColor = lipgloss.AdaptiveColor{Dark: "#293229", Light: "#C8E6C9"}
	theme.DiffRemovedLineNumberBgColor = lipgloss.AdaptiveColor{Dark: "#332929", Light: "#FFCDD2"}
	theme.MarkdownTextColor = lipgloss.AdaptiveColor{Dark: darkForeground, Light: lightForeground}
	theme.MarkdownHeadingColor = lipgloss.AdaptiveColor{Dark: darkSecondary, Light: lightSecondary}
	theme.MarkdownLinkColor = lipgloss.AdaptiveColor{Dark: darkPrimary, Light: lightPrimary}
	theme.MarkdownLinkTextColor = lipgloss.AdaptiveColor{Dark: darkCyan, Light: lightCyan}
	theme.MarkdownCodeColor = lipgloss.AdaptiveColor{Dark: darkGreen, Light: lightGreen}
	theme.MarkdownBlockQuoteColor = lipgloss.AdaptiveColor{Dark: darkYellow, Light: lightYellow}
	theme.MarkdownEmphColor = lipgloss.AdaptiveColor{Dark: darkYellow, Light: lightYellow}
	theme.MarkdownStrongColor = lipgloss.AdaptiveColor{Dark: darkAccent, Light: lightAccent}
	theme.MarkdownHorizontalRuleColor = lipgloss.AdaptiveColor{Dark: darkComment, Light: lightComment}
	theme.MarkdownListItemColor = lipgloss.AdaptiveColor{Dark: darkPrimary, Light: lightPrimary}
	theme.MarkdownListEnumerationColor = lipgloss.AdaptiveColor{Dark: darkCyan, Light: lightCyan}
	theme.MarkdownImageColor = lipgloss.AdaptiveColor{Dark: darkPrimary, Light: lightPrimary}
	theme.MarkdownImageTextColor = lipgloss.AdaptiveColor{Dark: darkCyan, Light: lightCyan}
	theme.MarkdownCodeBlockColor = lipgloss.AdaptiveColor{Dark: darkForeground, Light: lightForeground}
	theme.SyntaxCommentColor = lipgloss.AdaptiveColor{Dark: darkComment, Light: lightComment}
	theme.SyntaxKeywordColor = lipgloss.AdaptiveColor{Dark: darkSecondary, Light: lightSecondary}
	theme.SyntaxFunctionColor = lipgloss.AdaptiveColor{Dark: darkPrimary, Light: lightPrimary}
	theme.SyntaxVariableColor = lipgloss.AdaptiveColor{Dark: darkRed, Light: lightRed}
	theme.SyntaxStringColor = lipgloss.AdaptiveColor{Dark: darkGreen, Light: lightGreen}
	theme.SyntaxNumberColor = lipgloss.AdaptiveColor{Dark: darkAccent, Light: lightAccent}
	theme.SyntaxTypeColor = lipgloss.AdaptiveColor{Dark: darkYellow, Light: lightYellow}
	theme.SyntaxOperatorColor = lipgloss.AdaptiveColor{Dark: darkCyan, Light: lightCyan}
	theme.SyntaxPunctuationColor = lipgloss.AdaptiveColor{Dark: darkForeground, Light: lightForeground}
	return theme
}

func init() {
	RegisterTheme("opencode", NewOpenCodeTheme())
}
