FROM python:3.8

WORKDIR /app

COPY ./ /app/

COPY ./requirements.txt /app/

RUN pip install -r ./requirements.txt

COPY ./flagids.py /app/

CMD python3 ./flagids.py
