# Eraser: Cleaning up Images from Kubernetes Nodes

Eraser helps you remove a set of images from all Kubernetes nodes in a cluster and checks if they are non-running. Eraser is intended to be used with a scanner that will generate an [ImageList](api/v1alpha1/imagelist_types.go) holding the list of specific images to remove (ex: vulnerable, over 1 week old, non-MCR etc.)

## How to Use

To get started, build and push the eraser image:
* make docker-build-eraser
* make docker-push-eraser

Then, in your cluster, generate the [CRDs](api/v1alpha1) and [controllers](controllers):
* make generate manifests
* make deploly
* make docker-build
* make docker-push

Next, create an [ImageList](api/v1alpha1/imagelist_types.go) and specify the images you would like to remove manually. (As the project develops, this will change to use scanner)
* kubectl apply -f config/samples/[eraser_v1alpha1_imagelist.yaml](config/samples/eraser_v1alpha1_imagelist.yaml) --namespace="eraser-system"

This should have triggered an [ImageJob](api/v1alpha1/imagejob_types.go) that will deploy [eraser](pkg/eraser/eraser.go) pods on every node to perform the removal given the list of images. 

To view the result of the removal:
* describe ImageList CR and look at status field:
    * kubectl describe ImageList -n eraser-system imagelist_sample

To view the result of the ImageJob eraser pods:
* find name of ImageJob: 
    * kubectl get ImageJob -n eraser-system
* describe ImageJob CR and look at status field:
    * kubectl describe ImageJob -n eraser-system [name of ImageJob]

## Developer Setup

### Design 
* [Design Documentation](https://docs.google.com/document/d/1Rz1bkZKZSLVMjC_w8WLASPDUjfU80tjV-XWUXZ8vq3I/edit?usp=sharing) 

### Testing
* [Unit tests](.github/workflows/workflow.yaml) 
* E2E test in progress


## Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.opensource.microsoft.com.

When you submit a pull request, a CLA bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., status check, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## Trademarks

This project may contain trademarks or logos for projects, products, or services. Authorized use of Microsoft 
trademarks or logos is subject to and must follow 
[Microsoft's Trademark & Brand Guidelines](https://www.microsoft.com/en-us/legal/intellectualproperty/trademarks/usage/general).
Use of Microsoft trademarks or logos in modified versions of this project must not cause confusion or imply Microsoft sponsorship.
Any use of third-party trademarks or logos are subject to those third-party's policies.
