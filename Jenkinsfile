// hello-world v3: Build -> Verify -> Deploy Staging (HTTPS) -> Approval -> Deploy Prod (simulasi) -> Promote
// Staging dan "prod" di sini sama-sama berjalan di srv-docker-dev (simulasi alur UAT -> persetujuan -> prod).
// Tag "stable" = versi yang sudah disetujui dan berjalan di "prod". Rollback selalu ke tag ini.
// Pemicu (Poll SCM) tetap diatur di job Jenkins.

// Uji respons lewat sidecar curl yang berbagi network container target (image akhir = scratch, tanpa curl/shell).
def smoke(String container, String expected = 'hello world') {
    sh """
        sleep 2
        OUT=\$(docker run --rm --network container:${container} curlimages/curl -fsS http://localhost:8080)
        echo "Response: \$OUT"
        echo "\$OUT" | grep -q "${expected}"
    """
}

// Uji dari luar lewat domain HTTPS (menguji Traefik + sertifikat). Mencoba ulang sampai ~3 menit
// karena sertifikat baru bisa butuh 1-2 menit untuk terbit pada deploy pertama.
def check(String url, String expected = 'hello world') {
    sh """
        OUT=""
        for i in \$(seq 1 30); do
            OUT=\$(curl -fsS --max-time 5 ${url} 2>/dev/null) && break
            sleep 6
        done
        echo "Response dari ${url}: \$OUT"
        echo "\$OUT" | grep -q "${expected}"
    """
}

// Jalankan container di network proxy dengan label Traefik (HTTPS lewat certresolver cf), tanpa port publik di host.
def deploy(String name, String host, String image) {
    sh """
        docker rm -f ${name} || true
        docker run -d --name ${name} --restart unless-stopped --network proxy \\
          --label traefik.enable=true \\
          --label traefik.docker.network=proxy \\
          --label 'traefik.http.routers.${name}.rule=Host(`${host}`)' \\
          --label traefik.http.routers.${name}.entrypoints=websecure \\
          --label traefik.http.routers.${name}.tls.certresolver=cf \\
          --label traefik.http.services.${name}.loadbalancer.server.port=8080 \\
          ${image}
    """
}

// Kembalikan container ke versi stable (versi terakhir yang disetujui). Jika belum ada, hanya hapus container gagal.
def rollback(String name, String host) {
    sh "docker logs ${name} || true"
    def hasStable = sh(script: "docker image inspect ${env.IMAGE}:stable >/dev/null 2>&1", returnStatus: true) == 0
    if (hasStable) {
        echo "Gagal, rollback ${name} ke versi stable"
        deploy(name, host, "${env.IMAGE}:stable")
    } else {
        sh "docker rm -f ${name} || true"
        echo 'Belum ada versi stable untuk rollback'
    }
}

pipeline {
  agent any

  options {
    timestamps()
    disableConcurrentBuilds()
    timeout(time: 45, unit: 'MINUTES')   // sudah termasuk waktu menunggu persetujuan
  }

  parameters {
    booleanParam(name: 'SIMULATE_PROD_FAILURE', defaultValue: false,
                 description: 'Khusus demo: paksa pengecekan deploy Prod gagal agar rollback terlihat')
  }

  environment {
    IMAGE     = 'kai/hello-world'
    NAME_STG  = 'hello-world'
    NAME_PRD  = 'hello-world-prod'
    HOST_STG  = 'hello.koperasi-astra.com'
    HOST_PRD  = 'hello-prod.koperasi-astra.com'
    APPROVERS = 'it-kai-admin'   // ID user Jenkins yang boleh menyetujui, pisahkan dengan koma (isi akun personal)
    KEEP      = '5'              // jumlah image bernomor build yang disimpan
  }

  stages {
    stage('Build') {
      steps {
        sh 'docker build -t $IMAGE:$BUILD_NUMBER .'
      }
    }

    // Uji versi baru di container kandidat (tanpa port publik dan tanpa Traefik). Staging/prod tidak disentuh.
    stage('Verify') {
      steps {
        sh 'docker rm -f $NAME_STG-candidate || true'
        sh 'docker run -d --name $NAME_STG-candidate $IMAGE:$BUILD_NUMBER'
        script { smoke("${env.NAME_STG}-candidate") }
      }
      post {
        failure { sh 'docker logs $NAME_STG-candidate || true' }
        cleanup { sh 'docker rm -f $NAME_STG-candidate || true' }
      }
    }

    // Staging untuk UAT/SIT: domain HTTPS, rollback ke stable jika gagal.
    stage('Deploy Staging') {
      steps {
        script {
          deploy(env.NAME_STG, env.HOST_STG, "${env.IMAGE}:${env.BUILD_NUMBER}")
          smoke(env.NAME_STG)
          check("https://${env.HOST_STG}")
        }
      }
      post {
        failure { script { rollback(env.NAME_STG, env.HOST_STG) } }
      }
    }

    // Menunggu persetujuan manual. Tanpa executor, jadi tidak menahan Jenkins selagi menunggu.
    stage('Approval') {
      agent none
      steps {
        script {
          def approver = ''
          timeout(time: 30, unit: 'MINUTES') {
            approver = input(
              message: "Staging https://${env.HOST_STG} sudah lolos uji. Deploy build #${env.BUILD_NUMBER} ke Prod (simulasi)?",
              ok: 'Setujui dan deploy',
              submitter: env.APPROVERS,
              submitterParameter: 'APPROVED_BY')
          }
          env.APPROVED_BY = approver
          echo "Disetujui oleh: ${approver}"
        }
      }
    }

    stage('Deploy Prod (simulasi)') {
      steps {
        script {
          deploy(env.NAME_PRD, env.HOST_PRD, "${env.IMAGE}:${env.BUILD_NUMBER}")
          smoke(env.NAME_PRD, params.SIMULATE_PROD_FAILURE ? 'TEKS_TIDAK_ADA' : 'hello world')
          check("https://${env.HOST_PRD}", params.SIMULATE_PROD_FAILURE ? 'TEKS_TIDAK_ADA' : 'hello world')
        }
      }
      post {
        failure { script { rollback(env.NAME_PRD, env.HOST_PRD) } }
      }
    }

    // Tandai versi ini "stable" hanya jika sudah disetujui dan lolos di Prod.
    stage('Promote') {
      steps {
        sh 'docker tag $IMAGE:$BUILD_NUMBER $IMAGE:stable'
        echo "Build #${env.BUILD_NUMBER} menjadi stable. Disetujui oleh ${env.APPROVED_BY}."
      }
    }
  }

  post {
    success {
      // Simpan $KEEP image terbaru (dan tag stable), hapus sisanya. Hanya menyentuh kai/hello-world.
      sh '''
        docker images $IMAGE --format '{{.Tag}}' | grep -E '^[0-9]+$' | sort -rn | tail -n +$((KEEP + 1)) | while read t; do
          docker rmi $IMAGE:$t || true
        done
        docker image prune -f
      '''
    }
  }
}
