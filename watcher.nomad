job "watcher" {
  datacenters = ["dc1"]
  type = "service"

  group "watcher" {
    count = 1

    constraint {
      distinct_hosts = true
    }

    task "ipfs-watcher" {
      driver = "docker"

      config {
        image = "whyrusleeping/ipfs-watcher"
      }

      resources {
        network {
          port "ipfswatcher" {
            static = 9898
          }
        }
      }
    }
  }
}
