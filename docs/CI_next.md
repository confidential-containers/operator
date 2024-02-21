## Introduction

The Continuous Integration (CI) of Confidential Containers (CoCo) since its release 0.1.0 has evolved without major design changes despite many problem being noticed along the way, some really slowing down the development/releases, but we kept applying fixes and small improvements for the sake of just supporting the growth of the community and the initial releases. With the emerging of the "merge to main" initiative and recent disruptive changes on the CI of Kata Containers on main branch (e.g. migration from Jenkins to Github Actions) our CI reached a major breakage point.

On the kick-off meeting of the recently created [CoCo CI working group](https://docs.google.com/document/d/1gVuJXZzdyZcBg0Vje6n3-uL-SYdgxHzzEw9moU2ycoE) we community seemed to agreed that this is the time to re-design the CI to fix the problems that has hurt our development process. This document aims to capture the CI vision that was brought on that meeting as well as on the subsequents. On the next sections you will 

## The current state of affairs

But what's actually the current CoCo's CI?

There are two CIs that are considered:

 1. Kata Containers CI where jenkins jobs are triggered when a pull request to kata-containers/kata-containers' CCv0 branch. Each job implement the entire workflow (from build to test) that will be described below.
 1. Operator CI where jenkins jobs are triggered when a pull request to confidential-containers/operator.

### Kata Containers CI for CoCo

Each Jenkins job run either on Azure VM or baremetal. It does:
 1. (step 1) Install build and test dependencies in the system
 1. (step 2) Build and/or fetch from caches the whole software stack under test. The final product is the kata-container's tarball. The installation method is simply uncompress the tarball into the system.
 1. (step 3) Install Kubernetes on the system
 1. (step 4) Run test suites

On every step below there are problems, being the most impactful on development:

 1. (step 1) Takes too long
    * This setup step usually take several minutes, particurly a problem when developers run locally the CI
 1. (step 2) Redundant builds and testing loosily binaries 
    * Every job build almost the same components. Despicte being a waste, this increase the odds of failure due to external issues (e.g. networking failure to fetch a package)
    * There is tested the binaries built as extracted from the tarball into the test system, rather than testing the runtime-payload installed via kata-deploy (or the operator)

### Operator CI

### Current Kata Containers CI on main

The current Kata Containers CI on main fixed some issues:
 * Build the whole stack exactly once, no more redundant builds

### The problems

This isn't an compreensive list but rather just the ones we will tackle on CI next:

## A vision for the CI next

It's out of scope the CI for other components of CoCo (e.g. guest-components) other than the operator and Kata Containers.