FROM python:3.10

WORKDIR /app

COPY ./ /app/

COPY ./requirements.txt /app/

RUN pip install -r ./requirements.txt

COPY ./flagids.py /app/

STOPSIGNAL SIGINT

CMD ["python3", "./flagids.py"]
