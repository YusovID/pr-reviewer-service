pipeline {
    agent any

    environment {
        // Имена образов
        REGISTRY_IMAGE_APP = "registry.gitlab.com/yusovid/pr-reviewer-service/app"
        REGISTRY_IMAGE_MIGRATOR = "registry.gitlab.com/yusovid/pr-reviewer-service/migrator"
        
        // ID секретов из Jenkins
        REGISTRY_CREDS_ID = "docker-registry-creds"
        SSH_CREDS_ID = "vps-ssh-key"
        
        // Настройки сервера
        SSH_HOST = "213.165.48.130" 
        SSH_USER = "deployer"
    }

    stages {
        stage('Test & Lint') {
            agent {
                docker {
                    image 'golang:1.25'
                    args '-u root -v /var/run/docker.sock:/var/run/docker.sock -v go-mod-cache:/go/pkg/mod -v go-build-cache:/root/.cache/go-build'
                }
            }
            steps {
                script {
                    sh 'go env -w GOPROXY=https://goproxy.io,direct' 
                    echo "--- Running Tests ---"
                    sh 'go test -race -tags=integration ./...'
                }
            }
        }

        stage('Build & Push Images') {
            steps {
                script {
                    docker.withRegistry('https://registry.gitlab.com', "${REGISTRY_CREDS_ID}") {
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
                    // Подготовка файловой системы и статических конфигов
                    sh """
                        set -e
                        SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
                        
                        echo "Deploying configs to ${SSH_HOST}..."

                        # 1. Создаем структуру папок
                        ssh \$SSH_OPTS ${SSH_USER}@${SSH_HOST} 'mkdir -p ~/pr-reviewer/config'
                        
                        # 2. Копируем статические файлы (compose, prometheus, config)
                        scp \$SSH_OPTS compose.yml ${SSH_USER}@${SSH_HOST}:~/pr-reviewer/
                        scp \$SSH_OPTS prometheus.yml ${SSH_USER}@${SSH_HOST}:~/pr-reviewer/
                        scp \$SSH_OPTS -r config/ ${SSH_USER}@${SSH_HOST}:~/pr-reviewer/
                    """
                    
                    // Загрузка .env файла
                    withCredentials([file(credentialsId: 'env-file-prod', variable: 'ENV_FILE_PATH')]) {
                        sh """
                            set -e
                            SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
                            
                            echo "Uploading .env file..."
                            # Копируем файл с агента Jenkins на VPS
                            scp \$SSH_OPTS \$ENV_FILE_PATH ${SSH_USER}@${SSH_HOST}:~/pr-reviewer/.env
                            
                            # Ставим права доступа (безопасность)
                            ssh \$SSH_OPTS ${SSH_USER}@${SSH_HOST} "chmod 600 ~/pr-reviewer/.env"
                        """
                    }

                    // Запуск Docker Compose
                    sh """
                        set -e
                        SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
                        ssh \$SSH_OPTS ${SSH_USER}@${SSH_HOST} '
                            set -e
                            cd ~/pr-reviewer
                            
                            # Экспортируем имена образов для подстановки в compose.yml
                            export APP_IMAGE=${REGISTRY_IMAGE_APP}:latest
                            export MIGRATOR_IMAGE=${REGISTRY_IMAGE_MIGRATOR}:latest
                            
                            echo "Pulling images..."
                            docker compose pull -q
                            
                            echo "Restarting services..."
                            # Сначала down, чтобы подхватить новые переменные окружения, если они менялись
                            docker compose down --remove-orphans
                            docker compose up -d
                            
                            echo "Cleaning up..."
                            docker image prune -f
                        '
                    """
                }
            }
        }
    }
}