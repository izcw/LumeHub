# Changelog

本文件记录面向用户的版本变更。格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。

## [1.0.0] - 2026-05-30

首个可部署版本。

### Added

- 公开摄影画廊：瀑布流 / 网格、预览、原图、链接复制
- 加密分类与查看密钥（`?k=`）
- 多账号登录、Passkey、扫码登录
- 在线上传、编辑、排序、回收站
- 分类与布局管理、存储配额
- 图片组：JPG 预览 + RAW 原文件下载

### Deployment

- 单二进制 + `www/` 静态前端，无需数据库
- 宝塔部署说明见 [README.md](README.md#宝塔部署)
