# :scissors: fq-split

A utility for splitting FASTQ files at the N-th position. Works on paired and single end sequencing experiments.

## Example

```sh
# Paired-end:
fq-split -r1 data/test_r1.fq.gz -r2 data/test_r2.fq.gz -out paired-end-test -n 30

# Single-end:
fq-split -se data/test_r1.fq.gz -out single-end-test -n 30
```

> Use `fq-split -h` for more info.

## Quick introduction

**Note:** if using paired-end reads your FASTQ files need to be in the same order. For example: if first read from R1 file is the _@aaaa/1_, then the first read from R2 need to be the _@aaaa/2_.

```sh
# ----- Inspect input data -----
$ zcat data/test_r1.fq.gz | head

#@NDX550345_RUO:57:H5JY2BGXF:4:11506:26494:17984/1
#TAAGACATCACGAAGCATCACAGGTCTATCACCCTATTAACCACTCACGGGAGCACTCCATGCATATGGTATTTACGTCTGGGCGGTATGCACGCGATAGCATAGCCAGACGCTCGAGCCGCACCACCCAATGACGCACTAACTGACTT
#+
#AA6AAEEEE/AE////E/EEA//EEE//EEAEA<EE/AEEE6EE//EA//E//A/6AA//E/<E//EEAEAEEE/E<A6AE///EEEE/EE/</A//EEE<EE/A</A//EEAE/</A/AA/A/<EAAA/AEA/<A<//AE//EA/6/A

# ----- Run the program -----
$ fq-split -r1 data/test_r1.fq.gz -r2 data/test_r2.fq.gz -n 10 -out test
$ ls *.fq.gz
#test_begin_R1.fq.gz  test_begin_R2.fq.gz  test_end_R1.fq.gz  test_end_R2.fq.gz

# ----- Inspect output data -----
$ zcat test_begin_R1.fq.gz | head -n 4
#@NDX550345_RUO:57:H5JY2BGXF:4:11506:26494:17984/1
#TAAGACATCACGAAGCATCA
#+
#AA6AAEEEE/AE////E/EE

$ zcat test_end_R1.fq.gz | head -n 4
#@NDX550345_RUO:57:H5JY2BGXF:4:11506:26494:17984/1
#CAGGTCTATCACCCTATTAACCACTCACGGGAGCACTCCATGCATATGGTATTTACGTCTGGGCGGTATGCACGCGATAGCATAGCCAGACGCTCGAGCCGCACCACCCAATGACGCACTAACTGACTT
#+
#A//EEE//EEAEA<EE/AEEE6EE//EA//E//A/6AA//E/<E//EEAEAEEE/E<A6AE///EEEE/EE/</A//EEE<EE/A</A//EEAE/</A/AA/A/<EAAA/AEA/<A<//AE//EA/6/A
```

## :floppy_disk: Install

Download the binary file from github releases. Then you can:
1. Call it from where you decompressed the .tar.gz of github release.

    Example: `/home/userName/Downloads/fq-spliter -r1 r1.fq.gz -r2 r2.fq.gz`

2. Put it in a directory listed in your _$PATH_ and call it like any other program.

    Example: `mv fq-spliter /usr/local/bin/` and then `fq-spliter -h`
