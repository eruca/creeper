package creeper

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/moxar/arithmetic"
)

var (
	rx_funName = regexp.MustCompile(`^[a-z$][a-zA-Z]{0,15}`)
)

type Fun struct {
	Raw  string
	Node *Node

	Name   string
	Params []string

	Selection *goquery.Selection
	Result    string

	PrevFun *Fun
	NextFun *Fun
}

func (f *Fun) Append(s string) (*Fun, *Fun) {
	f.NextFun = ParseFun(f.Node, s)
	f.NextFun.PrevFun = f
	return f, f.NextFun
}

func PowerfulFind(s *goquery.Selection, q string) *goquery.Selection {
	rx_selectPseudoEq := regexp.MustCompile(`:eq\(\d+\)`)
	if rx_selectPseudoEq.MatchString(q) {
		rs := rx_selectPseudoEq.FindAllStringIndex(q, -1)
		sel := s
		for _, r := range rs {
			iStr := q[r[0]+4 : r[1]-1]
			i64, _ := strconv.ParseInt(iStr, 10, 32)
			i := int(i64)
			sq := q[:r[0]]
			q = strings.TrimSpace(q[r[1]:])
			sel = sel.Find(sq).Eq(i)
		}
		if len(q) > 0 {
			sel = sel.Find(q)
		}
		return sel
	} else {
		return s.Find(q)
	}
}

func (f *Fun) PageBody() (*goquery.Document, error) {
	body, err := f.Node.Page.Body()
	if err != nil {
		return nil, err
	}
	r := strings.NewReader(body)
	return goquery.NewDocumentFromReader(r)
}

func (f *Fun) InitSelector() error {
	if f.Node.IsArray || f.Node.IndentLen == 0 || f.Node.Page != nil {
		doc, err := f.PageBody()
		if err != nil {
			return err
		}
		bud := PowerfulFind(doc.Selection, f.Params[0])
		if len(bud.Nodes) > f.Node.Index {
			f.Selection = PowerfulFind(doc.Selection, f.Params[0]).Eq(f.Node.Index)
		} else {
			f.Node.Page.Inc()
			f.Node.Reset()
			f.InitSelector()
		}
	} else {
		_, err := f.Node.ParentNode.Fun.Invoke()
		if err != nil {
			return err
		}
		f.Selection = PowerfulFind(f.Node.ParentNode.Fun.Selection, f.Params[0]).Eq(f.Node.Index)
	}
	return nil
}

func (f *Fun) Invoke() (string, error) {
	var err error
	switch f.Name {
	case "$":
		err = f.InitSelector()
	case "attr":
		f.Result, _ = f.PrevFun.Selection.Attr(f.Params[0])
	case "text":
		f.Result = f.PrevFun.Selection.Text()
	case "html":
		f.Result, err = f.PrevFun.Selection.Html()
	case "outerHTML":
		f.Result, err = goquery.OuterHtml(f.PrevFun.Selection)
	case "style":
		f.Result, _ = f.PrevFun.Selection.Attr("style")
	case "href":
		f.Result, _ = f.PrevFun.Selection.Attr("href")
	case "src":
		f.Result, _ = f.PrevFun.Selection.Attr("src")
	case "class":
		f.Result, _ = f.PrevFun.Selection.Attr("class")
	case "id":
		f.Result, _ = f.PrevFun.Selection.Attr("id")
	case "calc":
		v, err := arithmetic.Parse(f.PrevFun.Result)
		if err != nil {
			return "", err
		}
		n, _ := arithmetic.ToFloat(v)
		prec := 2
		if len(f.Params) > 0 {
			i64, err := strconv.ParseInt(f.Params[0], 10, 32)
			if err != nil {
				return "", err
			}
			prec = int(i64)
		}
		f.Result = strconv.FormatFloat(n, 'g', prec, 64)
	case "expand":
		rx, err := regexp.Compile(f.Params[0])
		if err != nil {
			return "", err
		}
		src := f.PrevFun.Result
		dst := []byte{}
		m := rx.FindStringSubmatchIndex(src)
		s := rx.ExpandString(dst, f.Params[1], src, m)
		f.Result = string(s)
	case "match":
		rx, err := regexp.Compile(f.Params[0])
		if err != nil {
			return "", err
		}
		rs := rx.FindAllStringSubmatch(f.PrevFun.Result, -1)
		if len(rs) > 0 && len(rs[0]) > 1 {
			f.Result = rs[0][1]
		}
	}
	if err != nil {
		return "", err
	}
	if f.NextFun != nil {
		return f.NextFun.Invoke()
	} else {
		return f.Result, nil
	}
}

func ParseFun(n *Node, s string) *Fun {
	fun := new(Fun)
	fun.Node = n
	fun.Raw = s

	sa := rx_funName.FindAllString(s, -1)
	fun.Name = sa[0]
	ls := s[len(sa[0]):]
	ps := []string{}
	p, pl := parseParams(ls)
	for k := range p {
		ps = append(ps, k)
	}
	if len(ps) > 0 {
		fun.Params = ps
	}
	ls = ls[pl+1:]
	if len(ls) > 0 {
		ls = ls[1:]
		fun.Append(ls)
	}

	return fun
}
