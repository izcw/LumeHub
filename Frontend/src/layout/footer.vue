<template>
  <footer class="site-footer" aria-label="站点底部信息">
    <div class="site-footer__inner">
      <img class="site-footer__logo" :src="footerLogo" alt="网站 logo" />
      <div class="bottom-footer bottom-footer--line2">
        <a href="https://www.zhangchengwei.work/article/blog-3" target="_blank" rel="noopener noreferrer" title="关于作者">
          关于作者
        </a>
        <span class="bottom-footer__sep" aria-hidden="true">|</span>
        <span>{{ siteCopyright }}</span>
      </div>
      <div class="bottom-footer bottom-footer--line3">
        <a
          href="http://www.beian.gov.cn/portal/registerSystemInfo?recordcode=45098102000465"
          target="_blank"
          rel="noopener noreferrer"
          title="桂公网安备 45098102000466号"
        >
          <img
            src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABQAAAAUCAMAAAC6V+0/AAAC2VBMVEUAAAD+/ODz6Kr//+PeqFfYrn3x167oxnyaaVzhs2ifaFXbrGLkvFnpyF7v2X/kwm3cp1nhsGfqw3rZqG3ntVzjrFPt3oDjvGnfr2fbnFGti3q0lH7ktoLryXn9v1T4znr/74bnvGz034v+2I/ktoDz6ZLkwY/Dfz7buoftzYbq2IPr0pjs3bLv6KPRrnbKhFv79ND488n/+dDZr4Lx38f/+cH/95f42oL7/97s2Y3++uzw1rvTk3DmuloAAHkBAm7uzWYAAGXktV3qvFr/0ljksE7fo0rWHxhrdocAAIAABHf143Pyy27w1GwGA2jtymHpwWDqxV/qyVyTeFrrwFflwFPislP+xVLpsErbmUfVkEbysETemUTpgj7ThT3XdTg5FDjdhTXWZTDaTCm7TCbTOCLXPiD9LA/QFg3UAwnOAQOEj5kcPpdyhZSptJEACJFpfo4AG44XMInFvYfTvIejmYSVkINyeoJzdoK9un6SjX7FrnwAEHp8enny2HjWwHjKtnhcX3jYzHeNhnfu2HWUjHWsonPNwnH70m9WTm8AAW//723pym3dtmn/0mbnxGa0o2ZeWWb8zGT/4mPtwmJuYmL/22D/vmB5ZGC9kF7/2l0MAF3uyFqnjVn4xFjYnli0mVi5i1jiqVfyyVbmtlbXkVNUOFPlvFLpt1LNrFKjfVLuvlBgHlDsuU/ouU9ONU/ov05ODk7/2E02Gk3jqkqEaUr/tUngjkf7n0bXikb6xERCJETdn0LckUG1gD/ooD3Ulj3jkz3TZT3WjjzOeDqBWDr3pDnglTlMADnbbTf2gjbkbzaTYDZpAjbplzTtcTTEazPXXzOeXzDscS3MPi38jizJWSrVSCrrXynzfCjVdCjZRyjTQCbFUiTlYCPXPSHLPSHWMR/wXh7iRh7GPh3PLBrSIRrWGhfMJxPGJxPRDBG/ABG2ABCxDg7BDAvEGArZAAbJAALPAABwPJeHAAAAPXRSTlMACEIaxqxp+/v59/X08Ozs6+fm5eTk4eDb0tHLx8TDwb68urKxq5+cmJWTjIN8b11cWkxIQUE1MCknIRwUAZA0XgAAAXFJREFUGNNigAJmNmkpNmYGZGDKynXp6NZVHKxGCDHGmQlnLty73NuWsJARrlV+ffDNzNTkrB15y5Shohacmydefeb+9HbGu+keu4UswYJmrZ5zH2U+fP4q9eONSvtmc5CYtVyV177HH9zvp7l/SmvqqlWwAgrqHSv1XHAq69qB0O0Z59vty1x0gIIa56rzp+5KeZIUf3Ln5Bl2xdFqQEHhRJ8c+5TY6w8u3jl9yK+o7rgEUBAw7iPbcu3epCfdjY9Nf+1XEB4nDhRU2TOppPDE2/cvXyRfOVvT0BGmCRSU2buoL9vL0XHjmtmOduWznLbIAgXZ5y896FMf7DFhZWNnxZT9ka58QEHD5W5Rt+ZN2xQe7dLtkhgVFsQEcr0Bb2jcYQdnV1sH35jIHn6wGIOJYFDgkpDAAH9nN1vffgGIoJJ3i5OTs2vAihg3W1sHb1VIaGrxrFu8OmJtRMicDf5iurAAtdFXZ5EUFWFR1DYG8wEUrIUMZ1ticAAAAABJRU5ErkJggg=="
            title="桂公网安备 45098102000466号"
            alt="桂公网安备 45098102000466号"
            width="10"
            height="10"
          />
          桂公网安备 45098102000466号
        </a>
        <span class="bottom-footer__sep" aria-hidden="true">|</span>
        <a href="https://beian.miit.gov.cn/" target="_blank" rel="noopener noreferrer" title="桂ICP备2022009640号-2">
          桂ICP备2022009640号-2
        </a>
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
import footerLogo from '@/assets/images/logo2.2-text.png'

const COPYRIGHT_START_YEAR = 2026
const currentYear = new Date().getFullYear()
const siteCopyright =
  currentYear <= COPYRIGHT_START_YEAR
    ? `©${COPYRIGHT_START_YEAR} LumeHub`
    : `©${COPYRIGHT_START_YEAR}-${currentYear} LumeHub`
</script>

<style scoped lang="scss">
.site-footer {
  width: 100%;
  margin-top: 60px;
  padding: clamp(8px, 2vw, 14px) clamp(14px, 5vw, 54px) clamp(20px, 4vw, 30px);
  box-sizing: border-box;
}

.site-footer__inner {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  color: #8b8b8b;
  font-size: 10px;
  line-height: 1.5;
}

.site-footer__logo {
  width: auto;
  height: 60px;
  object-fit: contain;
  user-select: none;
}

.bottom-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: 8px;
}

.bottom-footer--line2,
.bottom-footer--line3 {
  color: #8b8b8b;
}

.bottom-footer__sep {
  color: #c7c7c7;
  user-select: none;
}

.bottom-footer a {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: inherit;
  text-decoration: none;
  white-space: nowrap;
  transition: color 0.2s ease;

  &:hover {
    color: #5f5f5f;
  }
}

@media (max-width: 640px) {
  .site-footer__inner {
    font-size: 10px;
  }

  .site-footer__logo {
    height: 60px;
  }
}
</style>
