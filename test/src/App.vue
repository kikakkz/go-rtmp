<template>
  <div class="liveView">
    <h3>Vue Video.js Live Demo</h3>
    <video-player
      class="vjs-custom-skin"
      ref="videoPlayer"
      :options="playerOptions"
      @ready="onPlayerReadied"
      @timeupdate="onTimeupdate"
    ></video-player>
  </div>
</template>

<script>
export default {
  name: 'live',
  data () {
    return {
      initialized: false,
      streams: {
        rtmp: 'rtmp://192.168.199.199/app/test'
      },
      playerOptions: {
        overNative: true,
        autoplay: false,
        controls: true,
        techOrder: ['flash'],
        sourceOrder: true,
        sources: [
          {
            type: 'rtmp/flv',
            src: 'rtmp://192.168.199.199/app/test'
          }
        ],
        poster: ''
      }
    }
  },
  methods: {
    onPlayerReadied () {
      if (!this.initialized) {
        this.initialized = true
      }
    },
    onTimeupdate (e) {
      console.log('currentTime', e.cache_.currentTime)
    }
  }
}
</script>

<style scoped>
.liveView {
  position: relative;
}
.selectWrapper {
  display: flex;
  justify-content: center;
  align-items: center;
  font-size: 20px;
  line-height: 30px;
  margin: 20px;
  vertical-align: baseline;
}
.inputWrapper {
  width: 500px;
  margin: 0 auto;
}
.buttonWrapper {
  text-align: center;
}
</style>
