package javadocset

import(
    "fmt"
    "os"
    "errors"
    "code.google.com/p/go-html-transform/h5"
    "code.google.com/p/go.net/html"
    "code.google.com/p/go-html-transform/css/selector"
    "path/filepath"
    "strings"
)

func main() {
    fmt.Println("w00t")
}

func Build(javadocPath string, docsetPath string) error {
    if exists, err := pathExists(javadocPath); !exists {
        return errors.New("Javadoc path does not exist")
    } else if err != nil {
        return err
    }

    if exists, err := pathExists(docsetPath); exists {
        return errors.New("Docset output path should not exist")
    } else if err != nil {
        return err
    }

    indexFile, err := os.Open(filepath.Join(javadocPath, "index-all.html"))
    if err != nil {
        return err
    }

    defer indexFile.Close()

    tree, err := h5.New(indexFile)
    if err != nil {
        return err
    }

    itemSelector, err := selector.Selector(".contentContainer dl dt")
    if err != nil {
        return err
    }

    anchorSelector, err := selector.Selector("a")
    if err != nil {
        return err
    }

    for _, node := range(itemSelector.Find(tree.Top())) {
        text := nodeText(node, false)
        anchor := anchorSelector.Find(node)[0]
        itemType := ""
        itemName := ""

        switch {
        case strings.Contains(text, "Class in"):
            itemName = nodeText(anchor, true)
            itemType = "Class"
        }

        if itemType != "" {
            fmt.Println(itemType, itemName)
        }
        // fmt.Println(text, nodeAttr(anchor, "href"))
    }

    return nil
}

func nodeText(node *html.Node, deep bool) string {
    var text []string
    walk := node.FirstChild
    for walk != nil {
        if(walk.Type == html.TextNode) {
            text = append(text, walk.Data)
        } else if walk.Type == html.ElementNode && deep {
            text = append(text, nodeText(walk, true))
        }
        walk = walk.NextSibling
    }
    return strings.Join(text, "")
}

func nodeAttr(node *html.Node, key string) string {
    for _, attr := range(node.Attr) {
        if attr.Key == key {
            return attr.Val
        }
    }
    return ""
}

func pathExists(path string) (bool, error) {
    if _, err := os.Stat(path); err != nil {
        if(os.IsNotExist(err)) {
            return false, nil
        } else {
            return false, err
        }
    }
    return true, nil
}

