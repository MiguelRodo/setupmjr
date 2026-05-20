#!/bin/bash
sed -i 's/	if err := SetupHPCGit(); err != nil {/	return nil/' internal/hpc/hpc.go
