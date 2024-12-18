#!/bin/bash

# Очистка экрана и перемещение курсора в начало
clear_screen() {
    printf "\033[2J\033[H"
}

# Функция для красивого ASCII-лого
print_logo() {
    cat << "EOF"
  ____ _           _
 / ___| |__   __ _| |_
| |   | '_ \ / _` | __|
| |___| | | | (_| | |_
 \____|_| |_|\__,_|\__|
EOF
    echo -e "\e[32m"  # Установка зеленого цвета
}

# Функция для вывода данных о сервере
print_server_info() {
    echo -e "\n🚀 \e[36mChamt Server Information 🚀\e[0m"
    echo -e "🌐 \e[33mAddress:\e[0m 127.0.0.1:8080"
    echo -e "📝 \e[33mName:\e[0m Chamt"
    echo -e "\e[34m======================================\e[0m"
}

# Функция для запуска Go-сервера
run_go_server() {
    echo -e "\n🔧 \e[35mStarting Go Server...\e[0m"
    go get
    go run main.go &
    SERVER_PID=$!

    # Ожидание завершения процесса
    wait $SERVER_PID
}

# Основная функция
main() {
    clear_screen
    print_logo
    print_server_info
    run_go_server
}

# Вызов основной функции
main
