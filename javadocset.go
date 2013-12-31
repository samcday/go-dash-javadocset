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
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
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

    resourcesDir := filepath.Join(docsetPath, "Contents", "Resources")

    if err := os.MkdirAll(resourcesDir, 0755); err != nil {
        return err
    }

    indexFile, err := os.Open(filepath.Join(javadocPath, "index-all.html"))
    if err != nil {
        return err
    }

    defer indexFile.Close()

    db, err := initDb(filepath.Join(resourcesDir, "docSet.dsidx"))
    if err != nil {
        return err
    }

    defer db.Close()

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

        switch {
        case strings.Contains(text, "Class in"):
            itemType = "Class"
        case strings.Contains(text, "Static method in"):
            itemType = "Method"
        case strings.Contains(text, "Static variable in"):
            itemType = "Field"
        case strings.Contains(text, "Constructor"):
            itemType = "Constructor"
        case strings.Contains(text, "Method in"):
            itemType = "Method"
        case strings.Contains(text, "Variable in"):
            itemType = "Field"
        case strings.Contains(text, "Interface in"):
            itemType = "Interface"
        case strings.Contains(text, "Exception in"):
            itemType = "Exception"
        case strings.Contains(text, "Error in"):
            itemType = "Error"
        case strings.Contains(text, "Enum in"):
            itemType = "Enum"
        case strings.Contains(text, "package"):
            itemType = "Package"
        case strings.Contains(text, "Annotation Type"):
            itemType = "Notation"
        }

        tx, err := db.Begin()
        if err != nil {
            return err
        }

        statement, err := tx.Prepare("insert into searchIndex(name, type, path) VALUES(?, ?, ?)")
        if err != nil {
            return err
        }
        defer statement.Close()

        if itemType != "" {
            itemName := nodeText(anchor, true)
            _, err := statement.Exec(itemName, itemType, nodeAttr(anchor, "href"))
            if err != nil {
                return err
            }
            fmt.Println("Inserted!")
        }

        tx.Commit()
    }

    return nil
}

func initDb(path string) (*sql.DB, error) {
    db, err := sql.Open("sqlite3", path)
    if err != nil {
        return nil, err
    }

    _, err = db.Exec("CREATE TABLE searchIndex(id INTEGER PRIMARY KEY, name TEXT, type TEXT, path TEXT)")
    if err != nil {
        db.Close()
        return nil, err
    }

    return db, err
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

