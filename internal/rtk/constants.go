package rtk

type ContentFormat string

const (
	FormatUnknown      ContentFormat = "unknown"
	FormatPlainText    ContentFormat = "plain_text"
	FormatJSON         ContentFormat = "json"
	FormatXML          ContentFormat = "xml"
	FormatHTML         ContentFormat = "html"
	FormatMarkdown     ContentFormat = "markdown"
	FormatGitDiff      ContentFormat = "git_diff"
	FormatGitStatus    ContentFormat = "git_status"
	FormatGrep         ContentFormat = "grep"
	FormatFind         ContentFormat = "find"
	FormatLS           ContentFormat = "ls"
	FormatTree         ContentFormat = "tree"
	FormatBuildOutput  ContentFormat = "build_output"
	FormatLog          ContentFormat = "log"
	FormatReadNumbered ContentFormat = "read_numbered"
	FormatSearchList   ContentFormat = "search_list"
)

func (f ContentFormat) String() string {
	return string(f)
}
