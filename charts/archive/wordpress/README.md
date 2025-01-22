# k8x Wordpress Chart

This was the initial version of k8x. It was built around tsx, but was changed because the resulting tsx was not benefitial over normal objects.

jsx could have been better than objects, but that required some props to be children. For example

```jsx
export default (props) => (
    <ingress>
        <metadata name={props.name}></metadata>
        <spec>
            More spec children.
        </spec>
    </ingress>
)
```

Unfortunately we can only type props of a jsx expression, not children. `<metadata>` and `spec` are children and therefore not covered by type support, which is one of the main benefits of using k8x over helm for example. I decided to ditch jsx/tsx in favor of proper type support. See more in the tldraw uploaded/linked (second page, "why transpiling") in the k8x repository