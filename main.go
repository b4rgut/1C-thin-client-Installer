package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type msi struct {
	filename   string
	transforms string
	dir        string
}

func downloadInstaller(filepath, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(path, filepath.Clean(dest) + string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func makeDir(dir string) error {
	_, err := os.Stat(dir)
	if err != nil {
		return os.MkdirAll(dir, os.ModePerm)
	}
	return nil
}

func installMSI(msifile msi) error {
	cmd := exec.Command("msiexec", "/i", msifile.filename, "/quiet", fmt.Sprintf("TRANSFORMS=%s", msifile.transforms))

	cmd.Dir = msifile.dir

	return cmd.Run()
}

func removeTempDir(dir string) error {
	return os.RemoveAll(dir)
}

func pause() {
	fmt.Println("Можно закрыть установщик...")
	var b []byte = make([]byte, 1)
	os.Stdin.Read(b)
}

func main() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Printf("Ошибка получения абсолютного пути: %v", err)
		pause()
		return
	}

	// Ссылка на скачивание zip архива с тонким клиентом 1С
	url := "https://getfile.dokpub.com/yandex/get/https://disk.yandex.ru/d/xxxxxxxxxxxx"

	err = makeDir(filepath.Join(dir, "tempInstaller"))
	if err != nil {
		log.Printf("Ошибка создания временной директории: %v", err)
		pause()
		return
	}

	zipFilePath := filepath.Join(dir, "tempInstaller/installer.zip")
	destDir := filepath.Join(dir, "tempInstaller/extracted")

	fmt.Println("Скачивание архива ...")
	err = downloadInstaller(zipFilePath, url)
	if err != nil {
		log.Printf("Ошибка скачивания архива: %v", err)
		pause()
		return
	}
	fmt.Println("Архив скачан.")

	fmt.Println("Распаковка архива ...")
	err = unzip(zipFilePath, destDir)
	if err != nil {
		log.Printf("Ошибка распаковки архива: %v", err)
		pause()
		return
	}
	fmt.Println("Архив распакован.")

	msifile := msi{
		filename:   "1CEnterprise 8 Thin client (x86-64).msi", // Название MSI файла
		transforms: "1049.mst", // Название файла трансформации
		dir:        filepath.Join(destDir, "setuptc64_8_3_24_1586"), // Путь до директории с MSI файлом, изменить только название ZIP архива
	}

	err = installMSI(msifile)
	if err != nil {
		log.Printf("Ошибка установки MSI: %v", err)
		pause()
		return
	}
	fmt.Println("MSI установлен.")

	err = removeTempDir(filepath.Join(dir, "tempInstaller"))
	if err != nil {
		log.Println("Ошибка удаления временной директории: %w", err)
		pause()
		return
	}

	pause()
}
