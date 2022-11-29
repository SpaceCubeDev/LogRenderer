pipeline {
  agent any
  stages {
    stage('Test') {
      steps {
        sh 'go test -v ./...'
      }
    }

    stage('Build') {
      steps {
        sh 'find compiled -type f -delete && chmod +x build_compress.sh && ./build_compress.sh --nostrip --nocompress'
      }
    }

    stage('Archive') {
      steps {
        archiveArtifacts 'compiled/*'
      }
    }

  }
}