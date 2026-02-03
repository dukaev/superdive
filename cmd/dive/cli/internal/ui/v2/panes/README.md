# UI v2 Snapshot Testing

Этот каталог содержит snapshot тесты для UI компонентов v2. Snapshot тестирование гарантирует визуальную стабильность TUI компонентов.

## Зачем это нужно?

В TUI приложениях самая частая проблема — "я поправил отступ здесь, а развалилась верстка там". Snapshot тесты решают эту проблему:

1. **Гарантия визуальной стабильности**: Любое случайное изменение в отступах или верстке будет немедленно обнаружено
2. **Документация**: Golden-файлы служат примерами того, как должен выглядеть компонент в разных состояниях
3. **Безопасный рефакторинг**: Можно смело менять стили, зная что тесты покажут диффер

## Структура

```
panes/
├── filetree/
│   ├── pane.go
│   ├── pane_test.go           # Snapshot тесты
│   └── __snapshots__/
│       └── pane_test.snap     # Golden файлы
├── layers/
│   ├── pane.go
│   ├── pane_test.go
│   └── __snapshots__/
│       └── pane_test.snap
├── details/
│   ├── pane.go
│   ├── pane_test.go
│   └── __snapshots__/
│       └── pane_test.snap
└── image/
    ├── pane.go
    ├── pane_test.go
    └── __snapshots__/
        └── pane_test.snap
```

## Запуск тестов

### Запустить все snapshot тесты:
```bash
task unit
```

### Обновить golden файлы (после изменения UI):
```bash
task unit-update-snapshots
```

### Запустить тесты для конкретной панели:
```bash
go test -v ./cmd/dive/cli/internal/ui/v2/panes/filetree
```

### Обновить snapshot для конкретной панели:
```bash
go test -v ./cmd/dive/cli/internal/ui/v2/panes/filetree -update
```

## Написание snapshot тестов

Пример теста:

```go
func TestPane_View_Focused(t *testing.T) {
    // 1. Подготовка тестовых данных
    testData := testutils.LoadTestImage(t)

    // 2. Создание компонента
    pane := New(testData.TreeVM)
    pane.SetSize(50, 20)

    // 3. Изменение состояния
    pane.Update(FocusStateMsg{Focused: true})

    // 4. Сравнение с эталоном
    view := pane.View()
    snaps.MatchSnapshot(t, view)
}
```

## Лучшие практики

1. **Тестируйте разные состояния**: Unfocused, Focused, Different Sizes, Different Data
2. **Используйте описательные имена тестов**: `TestPane_View_Focused`, `TestPane_View_SmallWidth`
3. **Не тестируйте каждую комбинацию**: Фокусируйтесь на важных edge cases
4. **Обновляйте snapshot осознанно**: Каждый раз при обновлении проверяйте diff

## Когда обновлять golden файлы?

Обновляйте snapshot когда:
- ✅ Вы намеренно меняете стиль или верстку
- ✅ Вы добавляете новый фичу в UI
- ✅ Вы рефакторите и результат визуально идентичен

НЕ обновляйте когда:
- ❌ Вы случайно сломали верстку
- ❌ Вы не понимаете почему изменился вывод
- ❌ Тест падает на CI (это значит есть проблема)

## Troubleshooting

### Тест падает с "Snapshot not found"
Это нормально при первом запуске. Запустите с `-update` чтобы создать snapshot.

### Тест падает с "Snapshot mismatch"
1. Посмотрите на diff в выводе теста
2. Если изменение ожидаемое → запустите с `-update`
3. Если нет → исправьте код

### Проблемы с путями к файлам
Убедитесь что вы запускаете тесты из корня репозитория.

## Дополнительные ресурсы

- [go-snaps documentation](https://github.com/gkampitakis/go-snaps)
- [Bubbletea testing best practices](https://github.com/charmbracelet/bubbletea/tree/master/tutorials)
