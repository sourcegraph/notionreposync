```
var txt string
for c := node.FirstChild(); c != nil; c = c.NextSibling() {
    segment := c.(*ast.Text).Segment
    txt = txt + string(segment.Value(source))
}
r.c.AppendRichText(&notionapi.RichText{Text: &notionapi.Text{Content: txt}, Annotations: &notionapi.Annotations{Code: true}})
return ast.WalkSkipChildren, nil
```

```go
var txt string
for c := node.FirstChild(); c != nil; c = c.NextSibling() {
    segment := c.(*ast.Text).Segment
    txt = txt + string(segment.Value(source))
}
r.c.AppendRichText(&notionapi.RichText{Text: &notionapi.Text{Content: txt}, Annotations: &notionapi.Annotations{Code: true}})
return ast.WalkSkipChildren, nil
```
