// SPDX-License-Identifier: AGPL-3.0-only

// Коннектор gate4.ai «Файлы»: чтение, список, добавление в конец (append,
// без перезаписи) и запись (по умолчанию — с подтверждением, класс write)
// в папках из настройки roots. Доступ к файлам — только функциями хоста fs_*;
// хост раскрывает ~, .. и симлинки и не выпускает за пределы
// capabilities.files (= roots), поэтому выйти из папок коннектор не может.
//
// Лицензия: AGPL-3.0-only с дополнительным разрешением для хоста gate4ai
// (LICENSE-EXCEPTION).
package main

import (
	_ "embed"

	"github.com/gate4ai/sdk-go/connector"
)

//go:embed manifest.yaml
var manifestYAML []byte

func init() { connector.Register(manifestYAML, Files{}) }

func main() {}
