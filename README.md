# Cybernetix (c8x)
Deploy and manage typesafe apps to kubernetes

## Features:

- .env integration
  - C8X_MY_VARIABLE
- Automatic namespace handling
  - Auto create/upgrade namespaces
- Quickstart
  - `c8x init`
  - `npm install -D @c8x/wordpress`
  - `import {Wordpress} from "@c8x/wordpress"`
  - `c8x install chart.ts`
- Packaging/Versioning
  - `npm version patch -m "Upgrade to 1.0.1 for reasons"`
  - `npm pack @c8x/wordpress`
  - `npm publish --access=public`
- Typescript
- Single binary
- Safe sandboxing
- Proper IDE support
- Chart inspection
  - Render charts based on its input
- Reusable components
  - Props
- Hooks (todo)
  - `<Wordpress beforeInstall={slackMessage} afterInstall={slackMessage} onError={handleError} />`
  - `beforeInstall` `afterInstall` `onInstallError` `beforeUpdate` `afterUpdate` `onUpdateError` 

## Usage

```
c8x install <file>
c8x inspect <file>
c8x new <path>
c8x version
```

## Goals
Reuse existing infrastructure and code features for enhanced developer experience

## Non Goals
- Replace helm

## Why configuration as code?
- Mill uses scala as configuration and has a indepth article about it's advantages
  - https://mill-build.org/mill/depth/why-scala.html
- Pulumi takes it even further and specifies the whole infrastructure as code
  - https://www.pulumi.com/
- Spring does this since 2008 to avoid large xml configurations
  - https://docs.spring.io/spring-javaconfig/docs/1.0.0.M4/reference/htmlsingle/spring-javaconfig-reference.html
- Yoke uses a similar concept
  - https://github.com/yokecd/yoke?tab=readme-ov-file

## Helm differentiation

I feel like helm was built by the ops side of devops people. c8x is built by the dev side of devops people.

In general c8x is pretty similar to helm. It also took a lot of inspiration from it. But where helm is reinventing the wheel, c8x just falls back to already used mechanisms and infrastructure. (npm/typescript/configuration)

| Topic | helm     | c8x   |
| -------- |----------|-------| 
| Packaging | custom   | npm   |
| Templating | gotmpl   | js/ts |
| Configuration | --set servers.foo.port=80 | .env  |
| Scripting | custom   | js/ts |
| Code sharing | custom   | js/ts |

By custom I mean either a custom implementation, or a existing template language with limited or changed features.

## Terminology

### Component

A piece of reusable/configurable code, that is typically a kubernetes object, for example ingress, pod, service
