pipeline {
    agent any

    environment {
        // Имена образов
        REGISTRY_IMAGE_APP = "registry.gitlab.com/yusovid/pr-reviewer-service/app"
        REGISTRY_IMAGE_MIGRATOR = "registry.gitlab.com/yusovid/pr-reviewer-service/migrator"
        
        // ID секретов из Jenkins
        REGISTRY_CREDS_ID = "docker-registry-creds"
        SSH_CREDS_ID = "vps-ssh-key"
        
        // IP твоего сервера (задай здесь или в .env)
        SSH_HOST = "213.165.48.130" 
        SSH_USER = "deployer"
    }

    stages {
        stage('Test & Lint') {
            agent {
                docker {
                    image 'golang:1.25'
                    args '-v /var/run/docker.sock:/var/run/docker.sock -v go-mod-cache:/go/pkg/mod -v go-build-cache:/root/.cache/go-build'
                }
            }
            steps {
                script {
                    // Ускоряем скачивание модулей
                    sh 'go env -w GOPROXY=https://goproxy.io,direct' 
                    
                    echo "--- Running Tests ---"
                    // -v убрали, чтобы лог не был километровым, оставили только ошибки
                    sh 'go test -race -tags=integration ./...'
                }
            }
        }

        stage('Build & Push Images') {
            steps {
                script {
                    docker.withRegistry('https://registry.gitlab.com', "${REGISTRY_CREDS_ID}") {
                        // Используем sh для явного контроля над флагами сборки и кэшем
                        
                        echo "Building App..."
                        sh "docker build -t ${REGISTRY_IMAGE_APP}:latest -f cmd/pr-reviewer/Dockerfile ."
                        sh "docker push ${REGISTRY_IMAGE_APP}:latest"

                        echo "Building Migrator..."
                        sh "docker build -t ${REGISTRY_IMAGE_MIGRATOR}:latest -f cmd/migrator/Dockerfile ."
                        sh "docker push ${REGISTRY_IMAGE_MIGRATOR}:latest"
                    }
                }
            }
        }

        stage('Deploy to VPS') {
            steps {
                sshagent(credentials: ["${SSH_CREDS_ID}"]) {
                    sh """
                        # Отключаем проверку хоста, чтобы не добавлять в known_hosts вручную
                        SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
                        
                        echo "Deploying to ${SSH_HOST}..."

                        # 1. Создаем структуру папок
                        ssh \$SSH_OPTS ${SSH_USER}@${SSH_HOST} 'mkdir -p ~/pr-reviewer/config'
                        
                        # 2. Копируем конфиги
                        scp \$SSH_OPTS compose.yml ${SSH_USER}@${SSH_HOST}:~/pr-reviewer/
                        scp \$SSH_OPTS prometheus.yml ${SSH_USER}@${SSH_HOST}:~/pr-reviewer/
                        scp \$SSH_OPTS -r config/ ${SSH_USER}@${SSH_HOST}:~/pr-reviewer/

                        # 3. Передаем .env файл из секрета Jenkins
                        # Мы используем withCredentials, чтобы записать содержимое в файл
                    """
                    
                    // Безопасная вставка .env файла
                    withCredentials([string(credentialsId: 'env-file-prod', variable: 'ENV_CONTENT')]) {
                        sh """
                            SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
                            # Экранируем переменные для корректной передачи через SSH
                            ssh \$SSH_OPTS ${SSH_USER}@${SSH_HOST} "echo '\$ENV_CONTENT' > ~/pr-reviewer/.env"
                        """
                    }

                    // 4. Финальный деплой
                    sh """
                        SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
                        ssh \$SSH_OPTS ${SSH_USER}@${SSH_HOST} '
                            cd ~/pr-reviewer
                            
                            # Экспортируем переменные окружения для docker-compose
                            export APP_IMAGE=${REGISTRY_IMAGE_APP}:latest
                            export MIGRATOR_IMAGE=${REGISTRY_IMAGE_MIGRATOR}:latest
                            
                            # Логинимся, чтобы скачать свежие образы (если репо приватный)
                            # docker login ... (обычно настраивается на сервере один раз или тут через переменные)
                            
                            echo "Pulling images..."
                            docker compose pull -q
                            
                            echo "Restarting services..."
                            docker compose up -d --remove-orphans
                            
                            echo "Cleaning up..."
                            docker image prune -f
                        '
                    """
                }
            }
        }
    }
}