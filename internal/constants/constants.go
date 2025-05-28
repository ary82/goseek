package constants

const (
	SEARCH_API = "https://www.googleapis.com/customsearch/v1"
	UA         = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36"
)

const PROMPT = `You are an expert summarizing the answers based on the provided contents.

	Given the context as a sequence of references with a reference id in the format of a leading [x], please answer the following question:

{{ %s }}

In the answer, use format [link1], [link2], ..., [n] to mention the sources where the reference is used. 
	At the end of the answer, also give the legend, specifying which number represents which link. Don't give the legend of links that are not used for the answer and merge the duplicates, both in content and legend. It should be coherent

Please create the answer strictly related to the context.
	If the context has no information about the query, please write "No related information found in the context."

Here is the context:
{{ %s }}
`
