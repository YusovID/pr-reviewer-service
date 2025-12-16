pipeline {
    agent any

    environment {
        REGISTRY_IMAGE_APP = "registry.gitlab.com/yusovid/pr-reviewer-service/app"
        REGISTRY_IMAGE_MIGRATOR = "registry.gitlab.com/yusovid/pr-reviewer-service/migrator"
        REGISTRY_CREDS_ID = "docker-registry-creds"
        SSH_CREDS_ID = "vps-ssh-key"
        SSH_HOST = "213.165.48.130" 
    }

    stages {
        stage('Test') {
            agent {
                docker { 
                    image 'golang:1.25'
                    args '-v /var/run/docker.sock:/var/run/docker.sock' 
                }
            }
            steps {
                sh 'go test -v -race -tags=integration ./...'
            }
        }

        stage('Build & Push') {
            environment {
                DOCKER_BUILDKIT = '1'
            }
            steps {
                script {
                    docker.withRegistry('https://registry.gitlab.com', "${REGISTRY_CREDS_ID}") {
                        // Build
                        def appImage = docker.build("${REGISTRY_IMAGE_APP}:latest", "-f cmd/pr-reviewer/Dockerfile .")
                        def migratorImage = docker.build("${REGISTRY_IMAGE_MIGRATOR}:latest", "-f cmd/migrator/Dockerfile .")
                        
                        // Push
                        appImage.push()
                        migratorImage.push()
                    }
                }
            }
        }

        stage('Deploy') {
            steps {
                sshagent(["${SSH_CREDS_ID}"]) {
                    // Копируем файлы и запускаем
                    sh """
                        ssh -o StrictHostKeyChecking=no ${SSH_USER}@${SSH_HOST} 'mkdir -p ~/pr-reviewer'
                        scp -o StrictHostKeyChecking=no compose.yml ${SSH_USER}@${SSH_HOST}:~/pr-reviewer/
                        scp -o StrictHostKeyChecking=no -r config ${SSH_USER}@${SSH_HOST}:~/pr-reviewer/
                        scp -o StrictHostKeyChecking=no prometheus.yml ${SSH_USER}@${SSH_HOST}:~/pr-reviewer/
                        
                        # Создаем .env (тут упрощенно, лучше через withCredentials)
                        # ssh ... 'echo "..." > .env'

                        ssh -o StrictHostKeyChecking=no ${SSH_USER}@${SSH_HOST} '
                            cd ~/pr-reviewer &&
                            export APP_IMAGE=${REGISTRY_IMAGE_APP}:latest &&
                            export MIGRATOR_IMAGE=${REGISTRY_IMAGE_MIGRATOR}:latest &&
                            docker compose pull &&
                            docker compose up -d'
                    """
                }
            }
        }
    }
}