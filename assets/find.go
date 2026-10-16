package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Galdoba/lazyam3/internal/domain/metadata"
)

func main() {
	// Флаги
	id := flag.Int64("id", 0, "ID элемента для поиска (обязательный)")
	file := flag.String("file", "metadata.json", "путь к json-файлу с метаданными")
	shallow := flag.Bool("shallow", false, "не включать потомков целевого объекта (только он сам и его родители)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Поиск элемента метаданных по id.\n\n")
		fmt.Fprintf(os.Stderr, "Использование:\n")
		fmt.Fprintf(os.Stderr, "  %s -id=<number> [-file=<path>] [-shallow]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Примеры:\n")
		fmt.Fprintf(os.Stderr, "  %s -id=42\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -id=42 -file=./data/catalog.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -id=42 -shallow\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Флаги:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Проверяем, что -id вообще был передан (0 — валидное значение id)
	idSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "id" {
			idSet = true
		}
	})
	if !idSet {
		fmt.Fprintln(os.Stderr, "ошибка: не задан обязательный параметр -id")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(2)
	}

	// Читаем файл
	data, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("не удалось прочитать файл %q: %v", *file, err)
	}

	// Десериализация
	var root metadata.Root
	if err := json.Unmarshal(data, &root); err != nil {
		log.Fatalf("не удалось распарсить json из %q: %v", *file, err)
	}

	// Выбираем режим
	descendants := metadata.WithDescendants
	if *shallow {
		descendants = metadata.WithoutDescendants
	}

	// Поиск
	found, ok := root.FindByID(*id, descendants)
	if !ok {
		fmt.Fprintf(os.Stderr, "элемент с id=%d не найден\n", *id)
		os.Exit(1)
	}

	// Вывод
	out, err := json.MarshalIndent(found, "", "  ")
	if err != nil {
		log.Fatalf("не удалось сериализовать результат: %v", err)
	}
	fmt.Println(string(out))
}
