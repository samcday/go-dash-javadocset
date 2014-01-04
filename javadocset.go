package javadocset

import (
	"code.google.com/p/go-html-transform/css/selector"
	"code.google.com/p/go-html-transform/h5"
	"code.google.com/p/go.net/html"
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3" // Included for sqlite3 driver support
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const plistTemplate = `
<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
  <dict>
    <key>CFBundleIdentifier</key>
    <string>{{.Name}}</string>
    <key>CFBundleName</key>
    <string>{{.Name}}</string>
    <key>DocSetPlatformFamily</key>
    <string>{{.Name}}</string>
    <key>dashIndexFilePath</key>
    <string>overview-summary.html</string>
    <key>DashDocSetFamily</key>
    <string>java</string>
    <key>isDashDocset</key>
    <true/>
  </dict>
</plist>`

// Build will construct a Java Docset for Dash, using the Javadoc contained in
// the javadocPath provided.
func Build(javadocPath string, docsetRoot, docsetName string) error {
	if exists, err := pathExists(javadocPath); !exists {
		return errors.New("javadoc path does not exist")
	} else if err != nil {
		return err
	}

	if exists, err := pathExists(docsetRoot); !exists {
		return errors.New("docset root path does not exist")
	} else if err != nil {
		return err
	}

	docsetPath := filepath.Join(docsetRoot, docsetName+".docset")
	contentsPath := filepath.Join(docsetPath, "Contents")
	resourcesDir := filepath.Join(contentsPath, "Resources")
	documentsDir := filepath.Join(resourcesDir, "Documents")
	if err := os.MkdirAll(documentsDir, 0755); err != nil {
		return err
	}

	if err := copyPath(javadocPath, documentsDir); err != nil {
		return err
	}

	plistPath := filepath.Join(contentsPath, "Info.plist")
	if err := writePlist(plistPath, docsetName); err != nil {
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

	itemSelector, err := selector.Selector("dl dt")
	if err != nil {
		return err
	}

	anchorSelector, err := selector.Selector("a")
	if err != nil {
		return err
	}

	for _, node := range itemSelector.Find(tree.Top()) {
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

func writePlist(path, docsetName string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return err
	}

	tmplData := map[string]string{
		"Name": docsetName,
	}

	return tmpl.Execute(file, tmplData)
}

func nodeText(node *html.Node, deep bool) string {
	var text []string
	walk := node.FirstChild
	for walk != nil {
		if walk.Type == html.TextNode {
			text = append(text, walk.Data)
		} else if walk.Type == html.ElementNode && deep {
			text = append(text, nodeText(walk, true))
		}
		walk = walk.NextSibling
	}
	return strings.Join(text, "")
}

func nodeAttr(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func pathExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func copyPath(src, dest string) error {
	src = filepath.Clean(src) + "/"
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		srcRel, _ := filepath.Rel(src, path)
		destPath := filepath.Join(dest, srcRel)

		if info.IsDir() {
			if err := os.Mkdir(destPath, info.Mode()); err != nil && !os.IsExist(err) {
				return err
			}
		} else {
			srcFile, err := os.Open(path)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE, info.Mode())
			if err != nil {
				return err
			}

			if _, err := io.Copy(destFile, srcFile); err != nil {
				destFile.Close()
				return err
			}

			return destFile.Close()
		}

		return nil
	})
}
