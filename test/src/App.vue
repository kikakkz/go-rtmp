<template>
  <div class="liveView">
    <h3>Vue Video.js Live Demo</h3>
    <video-player
      class="vjs-custom-skin"
      ref="videoPlayer"
      :options="playerOptions"
      @ready="onPlayerReady"
      @timeupdate="onTimeupdate"
    ></video-player>
  </div>
</template>

<script>
export default {
  name: 'live',
  data () {
    return {
      streams: {
        rtmp: 'rtmp://localhost/app/SampleVideo_1280x720_1mb.flv'
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
            src: 'rtmp://localhost/app/SampleVideo_1280x720_1mb.flv'
          }
        ],
        poster: ''
      }
    }
  },
  methods: {
    onPlayerReady () {
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
