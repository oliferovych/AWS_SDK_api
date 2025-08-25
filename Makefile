
all:
	cd lambda && make
	cd pulumi && pulumi up

fclean:
	cd lambda && make fclean
	cd pulumi && pulumi destroy --yes
